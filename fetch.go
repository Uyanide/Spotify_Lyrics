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
	Album        string
	Length       int // in ms
	IsLineSynced bool
	Is404        bool
	Lyrics       []LyricLine
}

// currently not used, track ID is enough since this program is called "spotify-"lyrics
func (data *LyricsData) formatName() string {
	f := func(str string) string {
		return strings.ReplaceAll(strings.TrimSpace(str), " ", "_")
	}
	return fmt.Sprintf("%s - %s - %s", f(data.Artist), f(data.Title), f(data.Album))
}

func NewLyricsDataCache(content string) (*LyricsData, error) {
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
		if currTime-int64(fetchTime) >= int64(REFETCH_INTERVAL_SEC) {
			return nil, fmt.Errorf("cached state expired, need to refetch")
		}
		// if not, avoid refetching
		return &LyricsData{
			Is404: true,
		}, nil
	}

	data := &LyricsData{}
	err := data.lrcDecodeLines(lines)
	if err != nil {
		return nil, fmt.Errorf("error decoding cached lyrics: %v", err)
	}
	return data, nil
}

func (data *LyricsData) _createErrorCache(cacheFile string) error {
	file, err := os.Create(cacheFile)
	if err != nil {
		return fmt.Errorf("error creating cache file: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	fmt.Fprintln(writer, "404")
	fmt.Fprintln(writer, time.Now().Unix()) // Store the fetch time
	writer.Flush()
	data.Is404 = true
	return nil
}

func (data *LyricsData) _createCache(cacheFile string) {
	if err := data.lrcEncodeFile(cacheFile); err != nil {
		log(fmt.Sprintf("Error creating cache file %s: %v", cacheFile, err))
	} else {
		log(fmt.Sprintf("Cached %d lines of lyrics at %s", len(data.Lyrics), cacheFile))
	}
}

func NewLyricsDataCurrentTrack(trackID string, cacheFile string) (*LyricsData, error) {
	ret := &LyricsData{}
	var err error
	// get length
	ret.Length, err = getLength()
	if err != nil {
		return nil, fmt.Errorf("error getting track length: %v", err)
	}
	// get metadata. if any of these fails, set to empty and ignore
	ret.Artist, err = getArtist()
	if err != nil {
		log(fmt.Sprintf("Error getting artist: %v", err))
		ret.Artist = ""
	}
	ret.Title, err = getTitle()
	if err != nil {
		log(fmt.Sprintf("Error getting title: %v", err))
		ret.Title = ""
	}
	ret.Album, err = getAlbum()
	if err != nil {
		log(fmt.Sprintf("Error getting album: %v", err))
		ret.Album = ""
	}
	// fetch from API
	for i := 0; i < RETRY_TIMES; i++ {
		err = ret.getLyricsLrclib()
		if err == nil {
			break
		}
		log(fmt.Sprintf("Error fetching lyrics (attempt %d/%d): %v", i+1, RETRY_TIMES, err))
		time.Sleep(time.Duration(RETRY_INTERVAL_SEC) * time.Second) // wait before retrying
	}
	if err != nil {
		log(fmt.Sprintf("Failed to fetch lyrics after %d attempts: %v", RETRY_TIMES, err))
		if err := ret._createErrorCache(cacheFile); err != nil {
			log(fmt.Sprintf("Error creating error cache: %v", err))
		}
		return nil, err
	}
	appendFetchLog(trackID, "Fetched lyrics successfully")
	ret._createCache(cacheFile)
	return ret, nil
}

func fetchLyrics(cacheDir string) (*LyricsData, error) {
	trackID, err := getTrackID()
	if err != nil {
		return nil, fmt.Errorf("error getting track ID: %v", err)
	}

	log(fmt.Sprintf("Fetching lyrics for track ID: %s", trackID))

	cacheFile := filepath.Join(cacheDir, trackID+".lrc")

	// Check cache first
	if content, err := os.ReadFile(cacheFile); err == nil {
		log(fmt.Sprintf("Cache hit for track ID: %s", trackID))
		ret, err := NewLyricsDataCache(string(content))
		if err != nil {
			log(fmt.Sprintf("Error parsing cached lyrics: %v", err))
			// ignore cache error, will fetch from API
		} else {
			return ret, nil
		}
	}

	// Fetch from API
	return NewLyricsDataCurrentTrack(trackID, cacheFile)
}

// get LyricsData from Spotify API, deprecated
func NewLyricsDataApi(trackID string) (*LyricsData, error) {
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
