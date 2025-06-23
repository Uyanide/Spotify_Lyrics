package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

func _onTrackChange(currRes *FetchResult, display *Display, nextIdx *int, trackID string, cacheDir string) {
	log(fmt.Sprintf("Switching to track ID: %s", trackID))
	display.Clear()
	*nextIdx = 0

	trackInfo, err := getTrackInfo()
	if err != nil {
		log(fmt.Sprintf("Error getting track info: %v", err))
		return
	}

	result, err := fetchLyrics(trackID, cacheDir)
	if err != nil || result == nil {
		display.AddLine(trackInfo)
		display.AddLine("No lyrics found")
		display.display()
		log(fmt.Sprintf("Error fetching lyrics for track ID %s: %v", trackID, err))
		return
	} else if result.IsInvalid {
		display.AddLine(trackInfo)
		display.AddLine("Lyrics not available")
		display.display()
		log(fmt.Sprintf("Lyrics for track ID %s not available", trackID))
		return
	} else if !result.IsSynced {
		display.AddLine(trackInfo)
		display.AddLine("Lyrics are not synced")
		display.display()
		log(fmt.Sprintf("Lyrics for track ID %s are not synced", trackID))
		return
	}

	*currRes = *result
}

func _listenProc(currTID string, currRes *FetchResult, display *Display, nextIdx *int, offset int, cacheDir string) {
	trackID, err := getTrackID()
	if currTID != trackID {
		currTID = trackID
		if err != nil {
			display.SingleLine("No track found")
			log(fmt.Sprintf("Error getting track ID: %v", err))
			return
		}
		_onTrackChange(currRes, display, nextIdx, trackID, cacheDir)
	}

	currPos, err := getPosition()
	if err != nil {
		display.SingleLine("Error getting position")
		log(fmt.Sprintf("Error getting position: %v", err))
		return
	}

	var changed bool
	for currRes != nil && *nextIdx < len(currRes.Lyrics) && currRes.Lyrics[*nextIdx].Time+offset <= currPos {
		display.AddLine(currRes.Lyrics[*nextIdx].Lyric)
		*nextIdx++
		changed = true
	}

	if changed {
		display.display()
	}
}

func listen(numLines int, offset int, cacheDir string, outputPath string, lockFile string) {
	lockFileHandle, err := acquireLock(lockFile)
	if err != nil {
		log(err.Error())
		os.Exit(1)
	}
	defer func() {
		syscall.Flock(int(lockFileHandle.Fd()), syscall.LOCK_UN)
		lockFileHandle.Close()
		os.Remove(lockFile)
	}()

	display := NewDisplay(numLines, outputPath)
	var currTID string
	var currRes *FetchResult
	var nextIdx int

	for {
		func() {
			defer func() {
				time.Sleep(config.INTERVAL)
			}()
			_listenProc(currTID, currRes, display, &nextIdx, offset, cacheDir)
		}()
	}
}

func print(numLines int, offset int, cacheDir string, outputPath string) {
	display := NewDisplay(numLines, outputPath)
	var currRes FetchResult
	var nextIdx int
	_listenProc("", &currRes, display, &nextIdx, offset, cacheDir)
}
