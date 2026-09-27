// Package out executes text-to-speech effects for the room runtime.
package out

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	renderSampleRate  = 24000
	pcmBytesPerSample = 2
	pcmMIME           = "audio/pcm"
)

// RenderLinesExecutor turns a pre-rendered text set into stored audio assets.
type RenderLinesExecutor struct {
	tts    ports.TTS
	assets ports.AssetWriter
}

// NewRenderLinesExecutor constructs a RenderLines executor.
func NewRenderLinesExecutor(tts ports.TTS, assets ports.AssetWriter) *RenderLinesExecutor {
	return &RenderLinesExecutor{tts: tts, assets: assets}
}

// Execute renders every text in effect and posts its terminal result to in.
// Rendering is ordered so the asset list in prerender_done matches effect.Texts.
func (e *RenderLinesExecutor) Execute(ctx context.Context, effect domain.RenderLines, scope domain.Scope, in ports.Inbox) {
	if err := e.validate(effect); err != nil {
		e.fail(ctx, effect.Set, scope, in)
		return
	}
	assets := make([]domain.Asset, 0, len(effect.Texts))
	for index, text := range effect.Texts {
		asset, err := e.render(ctx, text, effect.Voices[index], effect.Set)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			e.fail(ctx, effect.Set, scope, in)
			return
		}
		assets = append(assets, asset)
	}
	if ctx.Err() != nil {
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.PrerenderDone{Set: effect.Set, Assets: assets}})
}

func (e *RenderLinesExecutor) validate(effect domain.RenderLines) error {
	if e == nil || e.tts == nil || e.assets == nil {
		return errors.New("render lines dependencies are required")
	}
	if strings.TrimSpace(effect.Set) == "" || len(effect.Texts) == 0 || len(effect.Texts) != len(effect.Voices) {
		return errors.New("render lines requires matching non-empty text and voice sets")
	}
	for index, text := range effect.Texts {
		if strings.TrimSpace(text) == "" || strings.TrimSpace(effect.Voices[index]) == "" {
			return fmt.Errorf("render lines entry %d is empty", index)
		}
	}
	return nil
}

func (e *RenderLinesExecutor) render(ctx context.Context, text, voice, set string) (domain.Asset, error) {
	callMeta := ports.CallMeta{UtteranceID: domain.UtteranceID("prerender:" + set + ":" + inputHash(text, voice))}
	stream, err := e.tts.Stream(ctx, ports.TTSRequest{Meta: callMeta, VoiceID: voice, SampleRate: renderSampleRate}, &singleText{text: text})
	if err != nil {
		return domain.Asset{}, fmt.Errorf("start TTS: %w", err)
	}
	if stream == nil {
		return domain.Asset{}, errors.New("TTS returned a nil stream")
	}
	defer stream.Close()
	pcm, sampleRate, err := readPCM(stream)
	if err != nil {
		return domain.Asset{}, err
	}
	if len(pcm) == 0 {
		return domain.Asset{}, errors.New("TTS returned no PCM")
	}
	if sampleRate <= 0 {
		sampleRate = renderSampleRate
	}
	duration := len(pcm) * 1000 / (sampleRate * pcmBytesPerSample)
	meta := ports.AssetMeta{InputHash: inputHash(text, voice), DurationMS: duration}
	asset, err := e.assets.Write(ctx, vocab.AssetAudio, pcmMIME, pcm, meta)
	if err != nil {
		return domain.Asset{}, fmt.Errorf("store rendered audio: %w", err)
	}
	return asset, nil
}

func readPCM(stream ports.PCMStream) ([]byte, int, error) {
	var pcm []byte
	sampleRate := renderSampleRate
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return pcm, sampleRate, nil
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read TTS PCM: %w", err)
		}
		if chunk.SampleRate > 0 {
			sampleRate = chunk.SampleRate
		}
		pcm = append(pcm, chunk.S16LE...)
	}
}

func (e *RenderLinesExecutor) fail(ctx context.Context, set string, scope domain.Scope, in ports.Inbox) {
	if ctx.Err() != nil {
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.PrerenderFailed{Set: set}})
}

func post(ctx context.Context, in ports.Inbox, env domain.Envelope) {
	if in != nil {
		in.Post(ctx, env)
	}
}

func inputHash(text, voice string) string {
	sum := sha256.Sum256([]byte(voice + "\x00" + text))
	return hex.EncodeToString(sum[:])
}

type singleText struct {
	text string
	done bool
}

func (s *singleText) Recv() (string, error) {
	if s.done {
		return "", io.EOF
	}
	s.done = true
	return s.text, nil
}

func (s *singleText) Close() error { return nil }
