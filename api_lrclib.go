package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type LrclibLyricsResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
}

func (data *LyricsData) fetchLyricsLrclib() error {
	client := &http.Client{Timeout: FETCH_TIMEOUT}
	reqUrl := LRCLIB_API_URL +
		"?track_name=" + url.QueryEscape(data.Title) +
		"&artist_name=" + url.QueryEscape(data.Artist) +
		"&album_name=" + url.QueryEscape(data.Album) +
		"&duration=" + strconv.Itoa(data.Length/1000)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", LRCLIB_USER_AGENT)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			data.Is404 = true
		}
		return fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	var lrclibResp LrclibLyricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lrclibResp); err != nil {
		return fmt.Errorf("failed to parse lrclib response: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(lrclibResp.SyncedLyrics), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("invalid lrclib response format: no lines found")
	}
	err = data.lrcDecodeLines(lines)
	if data == nil || err != nil {
		return fmt.Errorf("failed to decode lyrics: %w", err)
	}
	data.IsLineSynced = true
	return nil
}
