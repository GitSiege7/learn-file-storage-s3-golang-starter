package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-movflags", "faststart", "-codec", "copy", "-f", "mp4", outputPath)

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to fast start file: %v", filePath)
	}

	return outputPath, nil
}
