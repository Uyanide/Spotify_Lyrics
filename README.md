A scripting-friendly small utility that talks directly to DBus, fetchs & caches & displays (as plain text) lyrics of the currently playing track from Spotify. This can also be used to partially control the player state of Spotify such as toggling play-pause and seeking to certain positions.

Currently included methods to get lyrics:

- cached files (`~/.cache/spotify_lyrcis/${trakid}.lrc` with an additional `[sync:]` label indicating the syncing type)
- fetching from Spotify (reimplemented [akashrchandran/spotify-lyrics-api](https://github.com/akashrchandran/spotify-lyrics-api));
- fetching from [LRCLIB](https://lrclib.net/).

> [!IMPORTANT]
>
> A `secret.go` (or whatever name) file containing `SP_DC` variable within `package main` should be created first in order to fetch lyrics from Spotify, which could look like:
>
> ```go
> package main
>
> var SP_DC = "SOME_RANDOM_STRING_FROM_SPOTIFY_COOKIES"
> ```

what this is for:

<figure>
    <img src="https://github.com/Uyanide/backgrounds/blob/master/screenshots/backdrop.jpg?raw=true"/>
    <figcaption>multiline lyrics at top-left & singleline lyrics at top-right</figcaption>
</figure>
