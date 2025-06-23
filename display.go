package main

import (
	"fmt"
	"os"
	"strings"
)

type Display struct {
	numLines   int
	lines      []string
	outputPath string
}

func NewDisplay(numLines int, outputPath string) *Display {
	if numLines < 1 {
		numLines = 1
	}
	return &Display{
		numLines:   numLines,
		lines:      make([]string, 0, numLines),
		outputPath: outputPath,
	}
}

func (d *Display) Clear() {
	d.lines = d.lines[:0]
	d.display()
}

func (d *Display) AddLine(line string) {
	if len(d.lines) >= d.numLines {
		d.lines = d.lines[1:]
	}
	d.lines = append(d.lines, line)
}

func (d *Display) display() {
	if err := os.WriteFile(d.outputPath, []byte(strings.Join(d.lines, "\n")+"\n"), 0644); err != nil {
		log(fmt.Sprintf("Error writing to output file: %v", err))
	}
}

func (d *Display) SingleLine(line string) {
	d.Clear()
	d.AddLine(line)
}

func log(message string) {
	fmt.Fprintf(os.Stderr, "%s\n", message)
}
