package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	defaultSampleRate = 16000
	frameSize         = 256
	hopSize           = 64
	minBPM            = 30.0
	maxBPM            = 170.0
)

const defaultTempoTolerance = 0.03

var errNoOnsets = errors.New("no regular onsets found")

// Measurement is the tempo and first-downbeat result for an audio take.
type Measurement struct {
	BPM        float64 `json:"bpm"`
	DownbeatMS float64 `json:"downbeat_ms"`
	SampleRate int     `json:"sample_rate"`
	Onsets     int     `json:"onsets"`
	Confidence float64 `json:"confidence"`
}

// AnalyzePCM estimates the tempo of signed little-endian mono PCM samples.
func AnalyzePCM(samples []int16, sampleRate int) (Measurement, error) {
	if sampleRate <= 0 {
		return Measurement{}, errors.New("sample rate must be positive")
	}
	if len(samples) < frameSize+hopSize {
		return Measurement{}, errors.New("audio is too short")
	}
	scores := energyScores(samples)
	onsets := findOnsets(scores, sampleRate)
	onsets = refineOnsets(samples, onsets, sampleRate)
	if len(onsets) < 2 {
		return Measurement{}, errNoOnsets
	}
	bpm, confidence := estimateTempo(onsets, sampleRate)
	if bpm == 0 {
		return Measurement{}, errNoOnsets
	}
	return Measurement{
		BPM:        bpm,
		DownbeatMS: float64(onsets[0]) * 1000 / float64(sampleRate),
		SampleRate: sampleRate,
		Onsets:     len(onsets),
		Confidence: confidence,
	}, nil
}

func foldTempo(measured, target, tolerance float64) (float64, error) {
	if measured <= 0 || target <= 0 {
		return 0, errors.New("tempo must be positive")
	}
	if tolerance <= 0 {
		return 0, errors.New("tempo tolerance must be positive")
	}
	for _, candidate := range []struct {
		tempo float64
		fold  float64
	}{
		{tempo: target, fold: 1},
		{tempo: target * 2, fold: 0.5},
		{tempo: target * 0.5, fold: 2},
	} {
		if math.Abs(measured-candidate.tempo) <= candidate.tempo*tolerance {
			return measured * candidate.fold, nil
		}
	}
	return 0, fmt.Errorf("BPM %.2f is outside %.2f BPM ± %.2f%%, including double and half time", measured, target, tolerance*100)
}

func refineOnsets(samples []int16, onsets []int, sampleRate int) []int {
	window := sampleRate / 50
	refined := make([]int, len(onsets))
	for i, onset := range onsets {
		start := onset - window
		if start < 0 {
			start = 0
		}
		end := onset + window
		if end > len(samples) {
			end = len(samples)
		}
		best := start
		for position := start + 1; position < end; position++ {
			if absSample(samples[position]) > absSample(samples[best]) {
				best = position
			}
		}
		refined[i] = best
	}
	return refined
}

func absSample(sample int16) int32 {
	value := int32(sample)
	if value < 0 {
		return -value
	}
	return value
}

func energyScores(samples []int16) []float64 {
	count := 1 + (len(samples)-frameSize)/hopSize
	scores := make([]float64, count)
	var previous float64
	for frame := 0; frame < count; frame++ {
		start := frame * hopSize
		var energy float64
		for _, sample := range samples[start : start+frameSize] {
			value := float64(sample) / 32768
			energy += value * value
		}
		energy = math.Sqrt(energy / frameSize)
		if energy > previous {
			scores[frame] = energy - previous
		}
		previous = energy
	}
	return scores
}

func findOnsets(scores []float64, sampleRate int) []int {
	if len(scores) < 3 {
		return nil
	}
	copyScores := append([]float64(nil), scores...)
	sort.Float64s(copyScores)
	baseline := copyScores[len(copyScores)*3/4]
	threshold := math.Max(baseline*0.25, 0.008)
	var onsets []int
	last := -int(0.2 * float64(sampleRate))
	for i := 1; i < len(scores)-1; i++ {
		if scores[i] < threshold || scores[i] < scores[i-1] || scores[i] < scores[i+1] {
			continue
		}
		position := i * hopSize
		if position-last < int(0.2*float64(sampleRate)) {
			if scores[i] > scores[(last)/hopSize] {
				onsets[len(onsets)-1] = position
				last = position
			}
			continue
		}
		onsets = append(onsets, position)
		last = position
	}
	return onsets
}

func estimateTempo(onsets []int, sampleRate int) (float64, float64) {
	if len(onsets) < 2 {
		return 0, 0
	}
	bestBPM, bestScore := 0.0, 0.0
	for bpm := minBPM; bpm <= maxBPM; bpm += 0.01 {
		period := float64(sampleRate) * 60 / bpm
		score := 0.0
		for i := 1; i < len(onsets); i++ {
			interval := float64(onsets[i] - onsets[i-1])
			beats := math.Round(interval / period)
			if beats < 1 || beats > 4 {
				continue
			}
			error := math.Abs(interval-beats*period) / period
			if error <= 0.04 {
				score += 1 - error/0.04
			}
		}
		if score > bestScore {
			bestBPM, bestScore = bpm, score
		}
	}
	possible := float64(len(onsets) - 1)
	return bestBPM, bestScore / possible
}
