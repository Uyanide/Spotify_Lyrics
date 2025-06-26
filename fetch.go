package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LyricLine struct {
	StartTimeMs string `json:"startTimeMs"`
	Words       string `json:"words"`
}

type LyricsResponse struct {
	SyncType string      `json:"syncType"`
	Lines    []LyricLine `json:"lines"`
}

type TimedLyric struct {
	Time  int
	Lyric string
}

type FetchResult struct {
	IsSynced  bool
	IsInvalid bool // Indicates if the lyrics were not found (404)
	Lyrics    []TimedLyric
}

func parseCachedLyrics(content string) (*FetchResult, error) {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid cached lyrics format: no lines found")
	} else if len(lines) < 2 || len(lines)%2 != 0 {
		return nil, fmt.Errorf("invalid cached lyrics format: expected even number of lines")
	}
	var result FetchResult

	if lines[0] == "404" {
		// check if cache is expired
		fetchTime, err := strconv.Atoi(lines[1])
		if err != nil {
			return nil, fmt.Errorf("invalid cached lyrics format: error parsing fetch time '%s': %v", lines[1], err)
		}
		currTime := time.Now().Unix()
		if currTime-int64(fetchTime) >= int64(config.REFETCH_INTERVAL) {
			return nil, fmt.Errorf("cached state expired, need to refetch")
		}
		// if not, treat as invalid
		result.IsInvalid = true
		return &result, nil
	}

	result.IsSynced = lines[0] == "SYNCED"

	result.Lyrics = make([]TimedLyric, 0, (len(lines)-1)/2)
	for i := 1; i < len(lines)-1; i += 2 {
		timeStr := strings.TrimSpace(lines[i])
		lyric := strings.TrimSpace(lines[i+1])

		if timeStr != "" {
			if time, err := strconv.Atoi(timeStr); err == nil {
				result.Lyrics = append(result.Lyrics, TimedLyric{Time: time, Lyric: lyric})
			} else {
				log(fmt.Sprintf("Error parsing time '%s': %v", timeStr, err))
				// skip
			}
		}
	}

	return &result, nil
}

func _fetchCacheError(cacheFile string) (*FetchResult, error) {
	file, err := os.Create(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("error creating cache file: %v", err)
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		fmt.Fprintln(writer, "404")
		fmt.Fprintln(writer, time.Now().Unix()) // Store the fetch time
		writer.Flush()
		var ret FetchResult
		ret.IsInvalid = true
		return &ret, nil
	}
}

func fetchLyrics(trackID string, cacheDir string) (*FetchResult, error) {
	log(fmt.Sprintf("Fetching lyrics for track ID: %s", trackID))

	cacheFile := filepath.Join(cacheDir, trackID+".txt")

	// Check cache first
	if content, err := os.ReadFile(cacheFile); err == nil {
		log(fmt.Sprintf("Cache hit for track ID: %s", trackID))
		res, err := parseCachedLyrics(string(content))
		if err != nil {
			log(fmt.Sprintf("Error parsing cached lyrics: %v", err))
			// ignore cache error, will fetch from API
		} else {
			return res, nil
		}
	}

	// Fetch from API
	resp, err := http.Get(fmt.Sprintf("%s?trackid=%s", config.APIUrl, trackID))
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			log(fmt.Sprintf("Track ID %s not found (404)", trackID))
			return _fetchCacheError(cacheFile)
		}
		return nil, fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	var lyricsResp LyricsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lyricsResp); err != nil {
		log(fmt.Sprintf("error decoding response: %v", err))
		// also regard as 404
		return _fetchCacheError(cacheFile)
	}

	var result FetchResult
	if lyricsResp.SyncType != "UNSYNCED" {
		result.IsSynced = true
	} else {
		result.IsSynced = false
	}

	if len(lyricsResp.Lines) == 0 {
		return nil, fmt.Errorf("no lyrics found")
	}
	result.Lyrics = make([]TimedLyric, 0, len(lyricsResp.Lines))

	// write track info as the first line
	trackInfo, err := getTrackInfo()
	if err != nil || trackInfo == "" {
		log(fmt.Sprintf("Error getting track info: %v", err))
		// ignore
	} else {
		result.Lyrics = append(result.Lyrics, TimedLyric{
			Time:  0,
			Lyric: trackInfo,
		})
	}

	// Cache the lyrics
	file, err := os.Create(cacheFile)
	if err != nil {
		log(fmt.Sprintf("Error creating cache file: %v", err))
		// ignore
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		if result.IsSynced {
			fmt.Fprintln(writer, "SYNCED")
		} else {
			fmt.Fprintln(writer, "UNSYNCED")
		}
		// first write track info
		if len(result.Lyrics) > 0 && result.Lyrics[0].Lyric != "" {
			fmt.Fprintln(writer, "0")
			fmt.Fprintln(writer, trackInfo)
		}
		// then write lyrics
		for _, line := range lyricsResp.Lines {
			fmt.Fprintf(writer, "%s\n%s\n", line.StartTimeMs, line.Words)
		}
		writer.Flush()
	}

	// Convert lyrics to TimedLyric
	for _, line := range lyricsResp.Lines {
		startTime, err := strconv.Atoi(line.StartTimeMs)
		if err != nil {
			log(fmt.Sprintf("Error parsing start time '%s': %v", line.StartTimeMs, err))
			continue
		}
		result.Lyrics = append(result.Lyrics, TimedLyric{Time: startTime, Lyric: line.Words})
	}

	log(fmt.Sprintf("Fetched %d lines of lyrics for track ID: %s", len(result.Lyrics), trackID))
	return &result, nil
}
