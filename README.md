currently included methods:

- fetching from Spotify (reimplemented [akashrchandran/spotify-lyrics-api](https://github.com/akashrchandran/spotify-lyrics-api));
- fetching from [LRCLIB](https://lrclib.net/).

A `secret.go` (or whatever preferred name) file containing `SP_DC` variable within `package main` should be created first in order to fetch lyrics from Spotify, which could look like:

```go
package main

var SP_DC = "SOME_RANDOM_STRING_FROM_SPOTIFY_COOKIES"
```
