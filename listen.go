package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

func (l *Listener) loop(interval int) {
	duration := time.Duration(interval) * time.Millisecond
	for {
		func() {
			l.proc()
			time.Sleep(duration)
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

	if l.currRes.IsInvalid || !l.currRes.IsSynced {
		return
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
		trackInfo = "Unknown Track"
	}

	result, err := fetchLyrics(l.currTID, l.cacheDir)
	if err != nil || result == nil {
		l.display.AddLine(trackInfo)
		l.display.AddLine("No lyrics found")
		l.display.display()
		l.currRes = FetchResult{
			IsInvalid: true,
		}
		return
	}
	l.currRes = *result
	if result.IsInvalid {
		l.display.AddLine(trackInfo)
		l.display.AddLine("Lyrics unavailable")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s unavailable", l.currTID))
	} else if !result.IsSynced {
		l.display.AddLine(trackInfo)
		l.display.AddLine("Lyrics unsynchronized")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s unsynced", l.currTID))
	}

}

func (l *Listener) getOffset() (int, error) {
	if l.offsetFile == "" {
		return l.offset, nil
	}
	content, err := os.ReadFile(l.offsetFile)
	if err != nil {
		log(fmt.Sprintf("Error reading offset file: %v", err))
		// If the file doesn't exist, create it with initial value 0
		if os.IsNotExist(err) {
			if err := os.WriteFile(l.offsetFile, []byte("0"), 0644); err != nil {
				return 0, fmt.Errorf("error creating offset file: %v", err)
			}
			log(fmt.Sprintf("Offset file created at %s with initial value 0", l.offsetFile))
			return 0, nil
		}
	}
	offset, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("error parsing offset from file: %v", err)
	}
	return offset, nil
}

func listen(numLines int, cacheDir string, outputPath string, lockFile string, offset int, offsetFile string, interval int) {
	if interval < config.MIN_LISTEN_INTERVAL {
		log(fmt.Sprintf("Minimum listen interval is %d milliseconds, using that instead", config.MIN_LISTEN_INTERVAL))
		interval = config.MIN_LISTEN_INTERVAL
	}
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

	(&Listener{
		display:    NewDisplay(numLines, outputPath),
		cacheDir:   cacheDir,
		offset:     offset,
		offsetFile: offsetFile,
	}).loop(interval)
}

func print(numLines int, cacheDir string, outputPath string, offset int, offsetFile string) {
	(&Listener{
		display:    NewDisplay(numLines, outputPath),
		cacheDir:   cacheDir,
		offset:     offset,
		offsetFile: offsetFile,
	}).proc()
}
