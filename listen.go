package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"
)

type Listener struct {
	display    *Display
	currTID    string
	currRes    FetchResult
	nextIdx    int
	currOffset int
	cacheDir   string
	offset     int
	offsetFile string
}

func (l *Listener) loop() {
	for {
		func() {
			l.proc()
			time.Sleep(config.LISTEN_INTERVAL)
		}()
	}
}

func (l *Listener) proc() {
	trackID, err := getTrackID()
	if l.currTID != trackID {
		l.currTID = trackID
		if err != nil {
			l.display.SingleLine("No track found")
			log(fmt.Sprintf("Error getting track ID: %v", err))
			return
		}
		l.onTrackChanged()
	}

	currPos, err := getPosition()
	if err != nil {
		l.display.SingleLine("Error getting position")
		log(fmt.Sprintf("Error getting position: %v", err))
		return
	}

	changed := false
	offset, err := l.getOffset()
	if err != nil {
		log(fmt.Sprintf("Error getting offset: %v", err))
	} else {
		l.currOffset = offset
	}
	log(fmt.Sprintf("Current position: %d, Offset: %d", currPos, offset))
	for l.nextIdx < len(l.currRes.Lyrics) && l.currRes.Lyrics[l.nextIdx].Time+l.currOffset <= currPos {
		l.display.AddLine(l.currRes.Lyrics[l.nextIdx].Lyric)
		l.nextIdx++
		changed = true
	}

	if changed {
		l.display.display()
	}
}

func (l *Listener) onTrackChanged() {
	log(fmt.Sprintf("Switching to track ID: %s", l.currTID))
	l.display.Clear()
	l.nextIdx = 0

	trackInfo, err := getTrackInfo()
	if err != nil {
		log(fmt.Sprintf("Error getting track info: %v", err))
		return
	}

	result, err := fetchLyrics(l.currTID, l.cacheDir)
	if err != nil || result == nil {
		l.display.AddLine(trackInfo)
		l.display.AddLine("No lyrics found")
		l.display.display()
		log(fmt.Sprintf("Error fetching lyrics for track ID %s: %v", l.currTID, err))
		return
	} else if result.IsInvalid {
		l.display.AddLine(trackInfo)
		l.display.AddLine("Lyrics not available")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s not available", l.currTID))
		return
	} else if !result.IsSynced {
		l.display.AddLine(trackInfo)
		l.display.AddLine("Lyrics are not synced")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s are not synced", l.currTID))
		return
	}

	l.currRes = *result
}

func (l *Listener) getOffset() (int, error) {
	if l.offsetFile != "" {
		content, err := os.ReadFile(l.offsetFile)
		if err != nil {
			return 0, fmt.Errorf("error reading offset file: %v", err)
		}
		offset, err := strconv.Atoi(string(content))
		if err != nil {
			return 0, fmt.Errorf("error parsing offset from file: %v", err)
		}
		return offset, nil
	}
	return l.offset, nil
}

func listen(numLines int, cacheDir string, outputPath string, lockFile string, offset int, offsetFile string) {
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

	listener := Listener{
		display:    NewDisplay(numLines, outputPath),
		currTID:    "",
		currRes:    FetchResult{},
		nextIdx:    0,
		currOffset: 0,
		cacheDir:   cacheDir,
		offset:     offset,
		offsetFile: offsetFile,
	}
	listener.loop()
}

func print(numLines int, cacheDir string, outputPath string, offset int, offsetFile string) {
	listener := Listener{
		display:    NewDisplay(numLines, outputPath),
		currTID:    "",
		currRes:    FetchResult{},
		nextIdx:    0,
		currOffset: 0,
		cacheDir:   cacheDir,
		offset:     offset,
		offsetFile: offsetFile,
	}
	listener.proc()
}
