package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func lrcDecodeLine(line string) (LyricLine, error) {
	re := regexp.MustCompile(`^\[(\d+):(\d+)\.(\d+)\](.*)$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 5 {
		return LyricLine{}, fmt.Errorf("invalid LRC line format: %s", line)
	}

	minutes, _ := strconv.Atoi(matches[1])
	seconds, _ := strconv.Atoi(matches[2])
	centiseconds, _ := strconv.Atoi(matches[3])
	lyrics := strings.TrimSpace(matches[4])

	startTimeMs := int(minutes*60000 + seconds*1000 + centiseconds*10)

	return LyricLine{
		StartTimeMs: startTimeMs,
		Words:       lyrics,
	}, nil
}

func lrcEncodeLine(line LyricLine) string {
	return fmt.Sprintf("[%02d:%02d.%02d]%s",
		line.StartTimeMs/60000,                               // minutes
		(line.StartTimeMs/1000)%60,                           // seconds
		int(math.Round(float64(line.StartTimeMs%1000)/10.0)), // 1/100 seconds
		line.Words)
}

func (data *LyricsData) lrcDecodeLines(lines []string) error {
	for _, line := range lines {
		if strings.HasPrefix(line, "[ti:") {
			data.Title = strings.TrimSuffix(strings.TrimPrefix(line, "[ti:"), "]")
		} else if strings.HasPrefix(line, "[ar:") {
			data.Artist = strings.TrimSuffix(strings.TrimPrefix(line, "[ar:"), "]")
		} else if strings.HasPrefix(line, "[al:") {
			data.Album = strings.TrimSuffix(strings.TrimPrefix(line, "[al:"), "]")
		} else if line == "[sync:line]" {
			data.IsLineSynced = true
		} else if line == "[sync:unknown]" {
			data.IsLineSynced = false
		} else if line != "" {
			lyricLine, err := lrcDecodeLine(line)
			if err != nil {
				log(fmt.Sprintf("error decoding line '%s': %v", line, err))
			} else {
				data.Lyrics = append(data.Lyrics, lyricLine)
			}
		}
	}
	return nil
}

func (data *LyricsData) lrcEncodeFile(path string) error {
	if data == nil {
		return nil
	}

	lines := make([]string, 0, len(data.Lyrics)+4)
	lines = append(lines, fmt.Sprintf("[ti:%s]", data.Title))
	lines = append(lines, fmt.Sprintf("[ar:%s]", data.Artist))
	lines = append(lines, fmt.Sprintf("[al:%s]", data.Album))
	if data.IsLineSynced {
		lines = append(lines, "[sync:line]")
	} else {
		lines = append(lines, "[sync:unknown]")
	}
	for _, lyric := range data.Lyrics {
		lines = append(lines, lrcEncodeLine(lyric))
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(content), 0644)
}
