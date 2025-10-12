package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type LyricsService struct {
	NumLines   int
	OutputPath string
	Cls        bool
	CacheDir   string
	Offset     int
	OffsetFile string
	Ahead      int

	display    *Display
	currTID    string
	currRes    LyricsData
	nextIdx    int
	currOffset int
	notFirst   bool
	prevPos    int
}

func (l *LyricsService) loop(interval int) {
	duration := time.Duration(interval) * time.Millisecond
	processing := make(chan struct{}, 1)

	for {
		select {
		case processing <- struct{}{}:
			go func() {
				defer func() {
					<-processing
				}()
				l.proc()
			}()
		default:
			log("Processing in progress, skipping")
		}
		time.Sleep(duration)
	}
}

func (l *LyricsService) proc() {
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

	if l.currRes.IsError || !l.currRes.IsLineSynced {
		// already handled in onTrackChanged
		return
	}

	currPos, err := getPosition()
	defer func() {
		l.prevPos = currPos
	}()
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
	// display first {ahead} lines
	if !l.notFirst {
		for i := 0; i < l.Ahead && i < len(l.currRes.Lyrics); i++ {
			l.display.AddLine(l.currRes.Lyrics[i].Words)
			changed = true
		}
		l.notFirst = true
	}
	if currPos < l.prevPos {
		// seek to the beginning if position moved backward
		// stupid but simple & effective :)
		l.display.Clear()
		l.nextIdx = 0
	}
	// seek forward
	for l.nextIdx < len(l.currRes.Lyrics) && l.currRes.Lyrics[l.nextIdx].StartTimeMs+l.currOffset <= currPos {
		if l.nextIdx+l.Ahead < len(l.currRes.Lyrics) {
			l.display.AddLine(l.currRes.Lyrics[l.nextIdx+l.Ahead].Words)
		} else {
			l.display.AddLine("")
		}
		l.nextIdx++
		changed = true
	}

	if changed {
		l.display.display()
	}
}

func (l *LyricsService) onTrackChanged() {
	log(fmt.Sprintf("Switching to track ID: %s", l.currTID))
	l.display.Clear()
	l.nextIdx = 0
	l.notFirst = false

	trackInfo := getTrackDisplayTitle()
	l.display.AddLine(trackInfo)

	result, err := fetchLyrics(l.CacheDir)
	if err != nil || result == nil {
		l.display.AddLine("No lyrics found")
		l.display.display()
		l.currRes = LyricsData{
			IsError: true,
		}
		return
	}
	l.currRes = *result
	if result.IsError {
		l.display.AddLine("Lyrics unavailable")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s unavailable", l.currTID))
	} else if !result.IsLineSynced {
		l.display.AddLine("Lyrics unsynchronized")
		l.display.display()
		log(fmt.Sprintf("Lyrics for track ID %s unsynced", l.currTID))
	} else if len(result.Lyrics) == 0 {
		l.display.AddLine("No lyrics found")
		l.display.display()
		log(fmt.Sprintf("No lyrics found for track ID %s", l.currTID))
	}
}

func (l *LyricsService) getOffset() (int, error) {
	if l.OffsetFile == "" {
		return l.Offset, nil
	}
	content, err := os.ReadFile(l.OffsetFile)
	if err != nil {
		log(fmt.Sprintf("Error reading offset file: %v", err))
		// If the file doesn't exist, create it with initial value 0
		if os.IsNotExist(err) {
			if err := os.WriteFile(l.OffsetFile, []byte("0"), 0644); err != nil {
				return 0, fmt.Errorf("error creating offset file: %v", err)
			}
			log(fmt.Sprintf("Offset file created at %s with initial value 0", l.OffsetFile))
			return 0, nil
		}
	}
	offset, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("error parsing offset from file: %v", err)
	}
	return offset, nil
}

func (s *LyricsService) listen(lockFile string, interval int) {
	if interval < MIN_LISTEN_INTERVAL_MS {
		log(fmt.Sprintf("Minimum listen interval is %d milliseconds, using that instead", MIN_LISTEN_INTERVAL_MS))
		interval = MIN_LISTEN_INTERVAL_MS
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

	s.display = NewDisplay(s.NumLines, s.OutputPath, s.Cls)
	s.loop(interval)
}

// 'print' is simply 'listen' without loops
func (s *LyricsService) print() {
	s.display = NewDisplay(s.NumLines, s.OutputPath, s.Cls)
	s.proc()
}
