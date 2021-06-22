Twitter Video Downloader
========================

Downloads videos from Twitter.

Building
--------

Either clone the repository into a local directory, or run
`go install https://github.com/vulppine/twitter-video-downloader`.

Usage
-----

Obtain a bearer token from Twitter to use for this application.
Set it as environmental variable `BEARER_TOKEN`, or put it into
a file named `token` (case-sensitive).

Afterwards, use it like so:

    twitter-video-downloader [twitter-id] (file-name)

License
-------

Copyright 2021 Flipp Syder under the MIT License
