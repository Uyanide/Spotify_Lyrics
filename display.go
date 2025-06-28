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
	tail       int
	size       int
	cls        bool
}

func NewDisplay(numLines int, outputPath string, cls bool) *Display {
	if numLines < 1 {
		log("Invalid number of lines, defaulting to 1")
		numLines = 1
	}
	return &Display{
		numLines:   numLines,
		lines:      make([]string, numLines),
		outputPath: outputPath,
		tail:       0,
		size:       0,
		cls:        cls,
	}
}

func (d *Display) Clear() {
	d.tail = 0
	d.size = 0
	d.display()
}

func (d *Display) AddLine(line string) {
	d.lines[d.tail%d.numLines] = line
	d.tail = (d.tail + 1) % d.numLines
	if d.size < d.numLines {
		d.size++
	}
}

func (d *Display) display() {
	builder := strings.Builder{}
	if d.cls {
		builder.WriteString("\033[H\033[2J") // Clear screen
	}
	head := d.tail + d.numLines - d.size
	// Fill empty lines
	for i := 0; i < d.numLines-d.size; i++ {
		builder.WriteString("\n")
	}
	for i := 0; i < d.size; i++ {
		builder.WriteString(d.lines[(head+i)%d.numLines] + "\n")
	}
	if err := os.WriteFile(d.outputPath, []byte(builder.String()), 0644); err != nil {
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
