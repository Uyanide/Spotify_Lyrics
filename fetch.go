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
	SyncType string      `json:"syncType"`
	Lines    []LyricLine `json:"lines"`
}

type FetchResult struct {
	IsSynced bool
	Is404    bool
	Lyrics   []LyricLine
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
		result.Is404 = true
		return &result, nil
	}

	result.IsSynced = lines[0] == "SYNCED"

	result.Lyrics = make([]LyricLine, 0, (len(lines)-1)/2)
	for i := 1; i < len(lines)-1; i += 2 {
		timeStr := strings.TrimSpace(lines[i])
		lyric := strings.TrimSpace(lines[i+1])

		if timeStr != "" {
			if time, err := strconv.Atoi(timeStr); err == nil {
				result.Lyrics = append(result.Lyrics, LyricLine{StartTimeMs: time, Words: lyric})
			} else {
				log(fmt.Sprintf("Error parsing time '%s': %v", timeStr, err))
				// skip
			}
		}
	}

	return &result, nil
}

func _createErrorCache(cacheFile string) (*FetchResult, error) {
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
		ret.Is404 = true
		return &ret, nil
	}
}

func _createCache(cacheFile string, res *FetchResult) {
	file, err := os.Create(cacheFile)
	if err != nil {
		log(fmt.Sprintf("Error creating cache file: %v", err))
		return // ignore cache error
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	if res.IsSynced {
		fmt.Fprintln(writer, "SYNCED")
	} else {
		fmt.Fprintln(writer, "UNSYNCED")
	}

	for _, lyric := range res.Lyrics {
		fmt.Fprintf(writer, "%d\n%s\n", lyric.StartTimeMs, lyric.Words)
	}
	writer.Flush()
	log(fmt.Sprintf("Cached %d lines of lyrics at %s", len(res.Lyrics), cacheFile))
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

func fetchAPI(trackID string) (*FetchResult, error) {
	resp, err := getLyrics(trackID)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, fmt.Errorf("no lyrics found for track ID: %s", trackID)
	}

	var result FetchResult
	log(fmt.Sprintf("Fetched lyrics for track ID: %s, sync type: %s\n", trackID, resp.Lyrics.SyncType))
	result.IsSynced = resp.Lyrics.SyncType == "SYNCED" || resp.Lyrics.SyncType == "LINE_SYNCED"
	result.Lyrics = make([]LyricLine, 0, len(resp.Lyrics.Lines)+1)

	title := getTrackInfo()
	result.Lyrics = append(result.Lyrics, LyricLine{
		StartTimeMs: 0,
		Words:       title,
	})

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
