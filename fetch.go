package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LyricLine struct {
	StartTimeMs int    `json:"startTimeMs"`
	Words       string `json:"words"`
}

type LyricsData struct {
	Artist       string
	Title        string
	IsLineSynced bool
	Is404        bool
	Lyrics       []LyricLine
}

func parseCachedLyrics(content string) (*LyricsData, error) {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("invalid cached lyrics format: no lines found")
	}

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
		return &LyricsData{
			Is404: true,
		}, nil
	}

	return lrcDecodeFile(lines)
}

func _createErrorCache(cacheFile string) (*LyricsData, error) {
	file, err := os.Create(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("error creating cache file: %v", err)
	} else {
		defer file.Close()
		writer := bufio.NewWriter(file)
		fmt.Fprintln(writer, "404")
		fmt.Fprintln(writer, time.Now().Unix()) // Store the fetch time
		writer.Flush()
		var ret LyricsData
		ret.Is404 = true
		return &ret, nil
	}
}

func _createCache(cacheFile string, res *LyricsData) {
	if err := lrcEncodeFile(cacheFile, res); err != nil {
		log(fmt.Sprintf("Error creating cache file %s: %v", cacheFile, err))
	} else {
		log(fmt.Sprintf("Cached %d lines of lyrics at %s", len(res.Lyrics), cacheFile))
	}
}

func fetchLyrics(trackID string, cacheDir string) (*LyricsData, error) {
	log(fmt.Sprintf("Fetching lyrics for track ID: %s", trackID))

	cacheFile := filepath.Join(cacheDir, trackID+".lrc")

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
	if res, err := fetchAPI(trackID); err != nil {
		appendFetchLog(trackID, fmt.Sprintf("Error fetching lyrics: %v", err))
		// also cache 404s
		if err.Error() == "404" {
			return _createErrorCache(cacheFile)
		}
		return nil, err
	} else {
		appendFetchLog(trackID, "Fetched lyrics successfully")
		_createCache(cacheFile, res)
		return res, nil
	}
}

func fetchAPI(trackID string) (*LyricsData, error) {
	resp, err := getLyrics(trackID)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, fmt.Errorf("no lyrics found for track ID: %s", trackID)
	}

	var result LyricsData
	log(fmt.Sprintf("Fetched lyrics for track ID: %s, sync type: %s\n", trackID, resp.Lyrics.SyncType))
	result.IsLineSynced = resp.Lyrics.SyncType == "SYNCED" || resp.Lyrics.SyncType == "LINE_SYNCED"
	result.Lyrics = make([]LyricLine, 0, len(resp.Lyrics.Lines)+1)

	result.Title, err = getTitle()
	if err != nil {
		result.Title = "UNKNOWN TITLE"
	}
	result.Artist, err = getArtist()
	if err != nil {
		result.Artist = "UNKNOWN ARTIST"
	}
	log(fmt.Sprintf("Track info: %s - %s", result.Artist, result.Title))

	for _, line := range resp.Lyrics.Lines {
		startTimeMs, err := strconv.Atoi(line.StartTimeMs)
		if err != nil {
			log(fmt.Sprintf("Error parsing start time '%s': %v", line.StartTimeMs, err))
			continue // skip this line if parsing fails
		}
		result.Lyrics = append(result.Lyrics, LyricLine{
			StartTimeMs: startTimeMs,
			Words:       line.Words,
		})
	}

	return &result, nil
}
