package main

import "time"

var (
	REFETCH_INTERVAL_SEC   = 300
	RETRY_INTERVAL_SEC     = 1
	RETRY_TIMES            = 3
	MIN_LISTEN_INTERVAL_MS = 50

	TOKEN_URL       = "https://open.spotify.com/api/token"
	LYRICS_URL      = "https://spclient.wg.spotify.com/color-lyrics/v2/track/"
	SERVER_TIME_URL = "https://open.spotify.com/api/server-time"
	SECRET_KEY_URL  = "https://raw.githubusercontent.com/Thereallo1026/spotify-secrets/refs/heads/main/secrets/secrets.json"

	USER_AGENT = "Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0" // some random UA from my current browser :)

	LRCLIB_API_URL    = "https://lrclib.net/api/get"
	LRCLIB_USER_AGENT = "github.com/Uyanide/spotify-lyrics"
	LRCLIB_TIMEOUT    = 30 * time.Second
)
