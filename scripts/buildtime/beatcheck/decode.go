package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
)

func decodeWithFFmpeg(ctx context.Context, input string) ([]int16, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-v", "error", "-i", input,
		"-f", "s16le", "-ac", "1", "-ar", fmt.Sprint(defaultSampleRate), "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create ffmpeg output: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}
	data, readErr := io.ReadAll(stdout)
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, fmt.Errorf("read ffmpeg output: %w", readErr)
	}
	if waitErr != nil {
		return nil, fmt.Errorf("ffmpeg decode: %w", waitErr)
	}
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("ffmpeg returned odd PCM byte count")
	}
	samples := make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return samples, nil
}
