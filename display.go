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
}

func NewDisplay(numLines int, outputPath string) *Display {
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
	head := d.tail + d.numLines - d.size
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
