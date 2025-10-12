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
	TrackID      string
	Artist       string
	Title        string
	Album        string
	Length       int // in ms
	IsLineSynced bool
	IsError      bool
	Is404        bool // no further refetching needed if 404 is received
	Lyrics       []LyricLine
}

// currently not used, track ID should be enough since this program is called "spotify-"lyrics
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

	isError := lines[0] == "error"
	is404 := lines[0] == "404"

	if isError || is404 {
		// check if cache is expired
		fetchTime, err := strconv.Atoi(lines[1])
		if err != nil {
			return nil, fmt.Errorf("invalid cached lyrics format: error parsing fetch time '%s': %v", lines[1], err)
		}
		currTime := time.Now().Unix()
		if (isError && currTime-int64(fetchTime) >= int64(REFETCH_INTERVAL_SEC)) ||
			(is404 && currTime-int64(fetchTime) >= int64(REFETCH_INTERVAL_SEC_404)) {
			return nil, fmt.Errorf("cached state expired, need to refetch")
		}
		// if not, avoid refetching by not returning an error
		return &LyricsData{
			IsError: isError,
			Is404:   is404,
		}, nil
	}

	data := &LyricsData{}
	err := data.lrcDecodeLines(lines)
	if err != nil {
		return nil, fmt.Errorf("error decoding cached lyrics: %v", err)
	}
	return data, nil
}

func (data *LyricsData) createErrorCache(cacheFile string) error {
	file, err := os.Create(cacheFile)
	if err != nil {
		return fmt.Errorf("error creating cache file: %v", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	var state string
	if data.Is404 {
		state = "404"
	} else {
		state = "error"
	}
	fmt.Fprintln(writer, state)
	fmt.Fprintln(writer, time.Now().Unix()) // Store the fetch time
	writer.Flush()
	data.IsError = true
	return nil
}

func (data *LyricsData) createCache(cacheFile string) {
	if err := data.lrcEncodeFile(cacheFile); err != nil {
		log(fmt.Sprintf("Error creating cache file %s: %v", cacheFile, err))
	} else {
		log(fmt.Sprintf("Cached %d lines of lyrics at %s", len(data.Lyrics), cacheFile))
	}
}

func NewLyricsDataCurrentTrack(cacheFile string) (*LyricsData, error) {
	ret := &LyricsData{}
	var err error

	// get track ID first
	ret.TrackID, err = getTrackID()
	if err != nil {
		return nil, fmt.Errorf("error getting track ID: %v", err)
	}
	// get length. 'crucial' according to lrclib.net
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

	// First try spotify API
	log("Fetching lyrics from Spotify API...")
	for i := 0; i < RETRY_TIMES; i++ {
		err = ret.fetchLyricsSpotify()
		if err == nil || ret.Is404 {
			break
		}
		log(fmt.Sprintf("Error fetching lyrics (attempt %d/%d): %v", i+1, RETRY_TIMES, err))
		time.Sleep(time.Duration(RETRY_INTERVAL_SEC) * time.Second) // wait before retrying
	}
	// If not successful or not synced, try lrclib.net
	if err != nil || !ret.IsLineSynced {
		if !ret.IsLineSynced {
			log("Fetched lyrics are not line-synced")
		}
		log("Fetching lyrics from lrclib.net...")
		for i := 0; i < RETRY_TIMES; i++ {
			err = ret.fetchLyricsLrclib()
			if err == nil || ret.Is404 {
				break
			}
			log(fmt.Sprintf("Error fetching lyrics from lrclib (attempt %d/%d): %v", i+1, RETRY_TIMES, err))
			time.Sleep(time.Duration(RETRY_INTERVAL_SEC) * time.Second) // wait before retrying
		}
	}

	if err != nil {
		log(fmt.Sprintf("Failed to fetch lyrics after %d attempts: %v", RETRY_TIMES, err))
		if err := ret.createErrorCache(cacheFile); err != nil {
			log(fmt.Sprintf("Error creating error cache: %v", err))
		}
		return nil, err
	}
	appendFetchLog(ret.TrackID, "Fetched lyrics successfully")
	ret.createCache(cacheFile)
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
	return NewLyricsDataCurrentTrack(cacheFile)
}
