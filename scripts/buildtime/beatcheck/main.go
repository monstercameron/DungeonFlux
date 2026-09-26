package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
)

func main() {
	input := flag.String("input", "", "audio file to decode with ffmpeg")
	expected := flag.Float64("bpm", 0, "expected BPM; reject outside tolerance when set")
	tolerance := flag.Float64("tolerance", 0.01, "relative BPM tolerance, default 1%%")
	flag.Parse()
	if *input == "" {
		fatal("-input is required")
	}
	if *tolerance <= 0 {
		fatal("-tolerance must be positive")
	}
	samples, err := decodeWithFFmpeg(context.Background(), *input)
	if err != nil {
		fatal(err.Error())
	}
	measurement, err := AnalyzePCM(samples, defaultSampleRate)
	if err != nil {
		fatal(err.Error())
	}
	if *expected > 0 && math.Abs(measurement.BPM-*expected) > *expected**tolerance {
		fatal(fmt.Sprintf("BPM %.2f is outside %.2f BPM ± %.2f%%", measurement.BPM, *expected, *tolerance*100))
	}
	encoded, err := json.MarshalIndent(measurement, "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	fmt.Println(string(encoded))
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "beatcheck:", message)
	os.Exit(1)
}
