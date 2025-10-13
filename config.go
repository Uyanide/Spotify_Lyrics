package main

import "time"

var (
	REFETCH_INTERVAL_SEC     = 300
	REFETCH_INTERVAL_SEC_404 = 3600 * 24 // 24 hours for 404
	RETRY_INTERVAL_SEC       = 1
	RETRY_TIMES              = 3
	MIN_LISTEN_INTERVAL_MS   = 50

	TOKEN_URL       = "https://open.spotify.com/api/token"
	LYRICS_URL      = "https://spclient.wg.spotify.com/color-lyrics/v2/track/"
	SERVER_TIME_URL = "https://open.spotify.com/api/server-time"
	SECRET_KEY_URL  = "https://raw.githubusercontent.com/xyloflake/spot-secrets-go/refs/heads/main/secrets/secrets.json"
	// SP_DC = "" // should be set elsewhere

	USER_AGENT        = "Mozilla/5.0 (X11; Linux x86_64; rv:143.0) Gecko/20100101 Firefox/143.0" // some random UA from my current browser :)
	USER_AGENT_HONEST = "spotify-lyrics (https://github.com/Uyanide/Spotify_Lyrics)"

	LRCLIB_API_URL = "https://lrclib.net/api/get"
	FETCH_TIMEOUT  = 30 * time.Second
)
