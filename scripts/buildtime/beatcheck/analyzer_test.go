package main

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzePCM_ClickTracks(t *testing.T) {
	for _, bpm := range []float64{80, 120, 170} {
		t.Run(formatBPM(bpm), func(t *testing.T) {
			samples := clickTrack(bpm, 16000, 12, 0.125)
			result, err := AnalyzePCM(samples, 16000)
			if err != nil {
				t.Fatalf("AnalyzePCM: %v", err)
			}
			if math.Abs(result.BPM-bpm) > bpm*0.005 {
				t.Fatalf("BPM = %.2f, want %.2f", result.BPM, bpm)
			}
			if math.Abs(result.DownbeatMS-125) > 8 {
				t.Fatalf("downbeat = %.1fms, want about 125ms", result.DownbeatMS)
			}
		})
	}
}

func TestAnalyzePCM_RejectsShortAndSilentAudio(t *testing.T) {
	for name, samples := range map[string][]int16{
		"short":  make([]int16, 10),
		"silent": make([]int16, 16000*2),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := AnalyzePCM(samples, 16000); err == nil {
				t.Fatal("AnalyzePCM succeeded")
			}
		})
	}
}

func TestAnalyzePCM_InvalidSampleRate(t *testing.T) {
	if _, err := AnalyzePCM(make([]int16, 1000), 0); err == nil {
		t.Fatal("AnalyzePCM succeeded")
	}
}

func TestHelpers_RefineAndEstimate(t *testing.T) {
	samples := make([]int16, 1000)
	samples[500] = -32000
	refined := refineOnsets(samples, []int{480}, 16000)
	if len(refined) != 1 || refined[0] != 500 {
		t.Fatalf("refined = %v, want [500]", refined)
	}
	if got, confidence := estimateTempo([]int{0, 8000, 16000}, 16000); math.Abs(got-120) > 0.01 || confidence <= 0 {
		t.Fatalf("estimateTempo = %.2f, %.2f", got, confidence)
	}
	if got, confidence := estimateTempo([]int{0}, 16000); got != 0 || confidence != 0 {
		t.Fatalf("short estimateTempo = %.2f, %.2f", got, confidence)
	}
	if got := findOnsets([]float64{0, 0.001, 0}, 16000); got != nil {
		t.Fatalf("findOnsets low signal = %v, want nil", got)
	}
	if absSample(-12) != 12 || absSample(12) != 12 {
		t.Fatal("absSample returned the wrong magnitude")
	}
}

func TestDecodeWithFFmpeg_StartFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", filepath.Join(dir, "missing"))
	if _, err := decodeWithFFmpeg(context.Background(), "input.wav"); err == nil {
		t.Fatal("decodeWithFFmpeg succeeded without ffmpeg")
	}
}

func TestDecodeWithFFmpeg_InvalidOutput(t *testing.T) {
	dir := t.TempDir()
	ffmpeg := filepath.Join(dir, "ffmpeg.cmd")
	if err := os.WriteFile(ffmpeg, []byte("@echo odd\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := decodeWithFFmpeg(context.Background(), "input.wav"); err == nil {
		t.Fatal("decodeWithFFmpeg accepted failed command")
	}
}

func clickTrack(bpm float64, sampleRate, beats int, firstSeconds float64) []int16 {
	period := int(math.Round(float64(sampleRate) * 60 / bpm))
	samples := make([]int16, int(float64(period*beats)+firstSeconds*float64(sampleRate))+sampleRate)
	first := int(firstSeconds * float64(sampleRate))
	for beat := 0; beat < beats; beat++ {
		start := first + beat*period
		for i := 0; i < sampleRate/100; i++ {
			if start+i >= len(samples) {
				break
			}
			samples[start+i] = int16(20000 * math.Exp(-float64(i)/25))
		}
	}
	return samples
}

func formatBPM(bpm float64) string {
	return map[float64]string{80: "80", 120: "120", 170: "170"}[bpm]
}
