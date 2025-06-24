package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func getTrackID() (string, error) {
	cmd := exec.Command("playerctl", "metadata", "mpris:trackid", "--player=spotify")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running playerctl: %v", err)
	}

	trackID := strings.TrimSpace(string(output))
	if trackID == "" {
		return "", fmt.Errorf("no track ID found")
	}

	parts := strings.Split(trackID, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid track ID format")
	}

	return parts[len(parts)-1], nil
}

func getPosition() (int, error) {
	cmd := exec.Command("playerctl", "position", "--player=spotify")
	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("error getting position: %v", err)
	}

	positionStr := strings.TrimSpace(string(output))
	position, err := strconv.ParseFloat(positionStr, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid position value: %v", err)
	}

	return int(position * 1000), nil // Convert to milliseconds
}

func getTrackInfo() (string, error) {
	cmd := exec.Command("playerctl", "metadata", "--format", "{{artist}} - {{title}}", "--player=spotify")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting track info: %v", err)
	}

	trackInfo := strings.TrimSpace(string(output))
	if trackInfo == "" {
		return "", fmt.Errorf("no track info found")
	}

	return trackInfo, nil
}

func getLength() (int, error) {
	cmd := exec.Command("playerctl", "metadata", "mpris:length", "--player=spotify")
	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("error getting track length: %v", err)
	}

	lengthStr := strings.TrimSpace(string(output))
	length, err := strconv.ParseInt(lengthStr, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid length value: %v", err)
	}

	return int(length / 1000), nil // Convert to milliseconds
}

func setPosition(position int) error {
	pos := float64(position) / 1000.0
	cmd := exec.Command("playerctl", "position", fmt.Sprintf("%.3f", pos), "--player=spotify")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error setting position: %v", err)
	}
	return nil
}
