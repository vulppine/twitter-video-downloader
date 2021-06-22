package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
)

// i kept all the old stuff in here to show the process of actually making
// this

// it went from:
// - let's try to figure out how to get videos from the v2 endpoint
// - ok, i found that there's a v1.1 API thing for videos for chunked streams via MPEG-DASH
// - wait, there's extended_entities in the v1.1 endpoint too???
// so in essence, it went from exciting to *really* boring once i found out about
// extended_entities

const tweetLookupEndpointV1 string = "https://api.twitter.com/1.1/statuses/show/"

// tweetLookupEndpoint string = "https://api.twitter.com/2/tweets/"
// tweetVideoEndpoint string = "https://api.twitter.com/1.1/videos/tweet/config/"

var (
	bearerToken string
	client      *http.Client
)

func requestWithToken(u string) *http.Request {
	r := new(http.Request)
	r.URL, _ = url.Parse(u)
	r.Header = make(http.Header)
	r.Header.Add(
		"Authorization",
		"Bearer "+bearerToken,
	)

	return r
}

// excluded a bunch of unneeded fields here
type tweetLookupV1Response struct {
	ExtendedEntities struct {
		Media []tweetExtendedMedia `json:"media"`
	} `json:"extended_entities"`
}

type tweetExtendedMedia struct {
	Type      string `json:"type"`
	VideoInfo struct {
		Variants tweetVideoInfoVariants `json:"variants"`
	} `json:"video_info"`
}

type tweetVideoInfoVariants []tweetVideoInfoVariant

func (t tweetVideoInfoVariants) Len() int           { return len(t) }
func (t tweetVideoInfoVariants) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t tweetVideoInfoVariants) Less(i, j int) bool { return t[i].Bitrate < t[j].Bitrate }

type tweetVideoInfoVariant struct {
	Bitrate     int    `json:"bitrate"`
	ContentType string `json:"content_type"`
	Url         string `json:"url"`
}

func tweetLookup(id string) (*tweetLookupV1Response, error) {
	resp, err := client.Do(requestWithToken(tweetLookupEndpointV1 + id + ".json"))
	if resp.StatusCode != 200 || err != nil {
		return nil, fmt.Errorf("error in retrieving tweet: status code %d, err %s", resp.StatusCode, err.Error())
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	t := new(tweetLookupV1Response)
	err = json.Unmarshal(b, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *tweetLookupV1Response) getBestVideo() ([]byte, error) {
	if len(t.ExtendedEntities.Media) == 0 {
		return []byte{}, fmt.Errorf("no media entities detected")
	}

	e := t.ExtendedEntities.Media[0]

	if e.Type != "video" {
		return []byte{}, fmt.Errorf("media entity is not of type video")
	}

	sort.Sort(t.ExtendedEntities.Media[0].VideoInfo.Variants)

	resp, err := http.Get(e.VideoInfo.Variants[len(e.VideoInfo.Variants)-1].Url)
	if resp.StatusCode != 200 || err != nil {
		return []byte{}, fmt.Errorf("error in retrieving video: status code %d, err %s", resp.StatusCode, err.Error())
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
}

/*
type tweetLookupResponse struct {
	Id       string        `json:"id"`
	Text     string        `json:"text"`
	AuthorId string        `json:"author_id"`
	Includes tweetIncludes `json:"includes"`
}

type tweetIncludes struct {
	Media []tweetMedia `json:"media"`
}

type tweetMedia struct {
	Type     string `json:"type"`
	MediaKey string `json:"media_key"`
}
*/

/*
type tweetVideo struct {
	// for this context, we need nothing but
	// the track and its url
	Track struct {
		Url string `json:"playbackUrl"`
	} `json:"track"`
}
*/

/*
func tweetLookup(id string, opts *url.Values) (*tweetLookupResponse, error) {
	u, err := url.Parse(tweetLookupEndpoint + id)
	if err != nil {
		return nil, err
	}

	if opts != nil {
		u.RawQuery = opts.Encode()
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	t := new(tweetLookupResponse)
	err = json.Unmarshal(b, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}
*/

/*
func getVideoMPD(id string) ([]byte, error) {
	resp, err := http.Get(tweetVideoEndpoint + id + ".json")
	if resp.StatusCode == 404 {
		return []byte{}, errors.New("tweet does not have video or invalid ID was given")
	} else if resp.StatusCode != 200 || err != nil {
		return []byte{}, errors.New("could not retrieve tweet")
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, errors.New("an error occurred reading the tweet")
	}

	v := new(tweetVideo)

	err = json.Unmarshal(b, v)
	if err != nil {
		return []byte{}, errors.New("an error occurred parsing the JSON endpoint response")
	}

	mpd, err := http.Get(v.Track.Url)
	if mpd.StatusCode != 200 || err != nil {
		return []byte{}, errors.New("an error occured getting the video MPD")
	}

	b, err = io.ReadAll(mpd.Body)
	if err != nil {
		return []byte{}, errors.New("an error occurred reading the video MPD")
	}

	return b, nil
}
*/

func main() {
	if os.Getenv("BEARER_TOKEN") == "" {
		if _, err := os.Stat("token"); !errors.Is(err, os.ErrNotExist) {
			f, _ := os.Open("token")
			b, err := io.ReadAll(f)
			if err != nil {
				panic(err)
			}

			bearerToken = strings.TrimRight(string(b), "\n")
		} else {
			fmt.Println("BEARER_TOKEN must be defined as an environmental variable, or in a file named 'token'.")
			os.Exit(1)
		}
	} else {
		bearerToken = os.Getenv("BEARER_TOKEN")
	}

	if os.Args[1] == "" {
		fmt.Println("The first argument must be a tweet ID. Aborting.")
	}

	client = new(http.Client)

	t, err := tweetLookup(os.Args[1])
	if err != nil {
		panic(err)
	}

	v, err := t.getBestVideo()
	if err != nil {
		panic(err)
	}

	var f *os.File

	if len(os.Args) != 3 {
		f, err = os.Create(os.Args[1] + ".mp4")
	} else {
		f, err = os.Create(os.Args[2] + ".mp4")
	}

	if err != nil {
		panic(err)
	}

	_, err = f.Write(v)
	if err != nil {
		panic(err)
	}

	f.Close()
}
