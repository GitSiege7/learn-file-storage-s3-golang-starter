package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type dimensions struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var buf bytes.Buffer
	cmd.Stdout = &buf

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run command, check file path: %v", filePath)
	}

	var dat dimensions
	err = json.Unmarshal(buf.Bytes(), &dat)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal data")
	}

	ls := 16.0 / 9.0
	pt := 9.0 / 16.0
	ratio := float64(dat.Streams[0].Width) / float64(dat.Streams[0].Height)

	if math.Abs(ratio-ls) < 0.05 {
		return "16:9", nil
	} else if math.Abs(ratio-pt) < 0.05 {
		return "9:16", nil
	} else {
		return "other", nil
	}
}
