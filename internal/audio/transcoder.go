package audio

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func Transcode(inputPath string, outputDir string) (outputPath string, durationSeconds int, err error) {
	filename := "output.m4a"

	outputFilePath := filepath.Join(outputDir, filename)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create outpit dir: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-t", "45",
		"-b:a", "128k",
		"-vn",
		outputFilePath,
	)

	if err = cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("failed to run cmd: %w", err)
	}

	//prints duration of audio file in seconds.
	probe := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		outputFilePath,
	)

	duration, err := probe.Output()
	if err != nil {
		return "", 0, fmt.Errorf("failed to probe file duration: %w", err)
	}

	secs, err := strconv.ParseFloat(strings.TrimSpace(string(duration)), 64)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	durationSeconds = int(math.Round(secs))

	return outputFilePath, durationSeconds, nil
}
