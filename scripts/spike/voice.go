package main

import (
	"context"
	"fmt"
	"io"
	"sync"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
)

// transcriber converts a complete MediaRecorder container into text.
type transcriber interface {
	Transcribe(context.Context, string, []byte) (string, error)
}

// fakeTranscriber makes local phone checks deterministic and free.
type fakeTranscriber struct{ text string }

func (t fakeTranscriber) Transcribe(context.Context, string, []byte) (string, error) {
	if len(t.text) == 0 {
		return "fake transcript", nil
	}
	return t.text, nil
}

// audioHub retains the latest fake PCM line for a DM listener.
type audioHub struct {
	mu      sync.RWMutex
	frames  []*spikev1.PCMFrame
	updated chan struct{}
}

func newAudioHub() *audioHub { return &audioHub{updated: make(chan struct{})} }

func (h *audioHub) publish(frames []*spikev1.PCMFrame) {
	h.mu.Lock()
	h.frames = clonePCM(frames)
	close(h.updated)
	h.updated = make(chan struct{})
	h.mu.Unlock()
}

func (h *audioHub) snapshot() ([]*spikev1.PCMFrame, <-chan struct{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return clonePCM(h.frames), h.updated
}

func clonePCM(frames []*spikev1.PCMFrame) []*spikev1.PCMFrame {
	out := make([]*spikev1.PCMFrame, len(frames))
	for i, frame := range frames {
		out[i] = &spikev1.PCMFrame{
			Sequence:   frame.GetSequence(),
			SampleRate: frame.GetSampleRate(),
			PcmS16Le:   append([]byte(nil), frame.GetPcmS16Le()...),
			Final:      frame.GetFinal(),
		}
	}
	return out
}

// voiceServer receives phone chunks and streams a short PCM fixture to the DM.
type voiceServer struct {
	spikev1.UnimplementedVoiceServer
	transcriber transcriber
	hub         *audioHub
}

func (s voiceServer) Talk(stream spikev1.Voice_TalkServer) error {
	var media []byte
	mediaType := "audio/webm"
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return s.finishTalk(stream.Context(), stream, mediaType, media)
		}
		if err != nil {
			return err
		}
		if len(media) == 0 && chunk.GetMimeType() != "" {
			mediaType = chunk.GetMimeType()
		}
		media = append(media, chunk.GetPcmS16Le()...)
		if chunk.GetFinal() {
			return s.finishTalk(stream.Context(), stream, mediaType, media)
		}
	}
}

func (s voiceServer) finishTalk(ctx context.Context, stream spikev1.Voice_TalkServer, mediaType string, media []byte) error {
	text, err := s.transcriber.Transcribe(ctx, mediaType, media)
	if err != nil {
		return fmt.Errorf("transcribe talk: %w", err)
	}
	s.hub.publish(fakePCM(text))
	return stream.SendAndClose(&spikev1.Transcript{Text: text})
}

func (s voiceServer) Listen(_ *spikev1.ListenRequest, stream spikev1.Voice_ListenServer) error {
	for {
		frames, changed := s.hub.snapshot()
		for _, frame := range frames {
			if err := stream.Send(frame); err != nil {
				return err
			}
		}
		if len(frames) > 0 {
			return nil
		}
		select {
		case <-changed:
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func fakePCM(text string) []*spikev1.PCMFrame {
	// A short audible 440 Hz tone proves the PCM path without a paid TTS call.
	const samples = 2400
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		value := int16(7000)
		if i%109 < 54 {
			value = -7000
		}
		pcm[i*2] = byte(value)
		pcm[i*2+1] = byte(value >> 8)
	}
	return []*spikev1.PCMFrame{{Sequence: 1, SampleRate: 24000, PcmS16Le: pcm, Final: true}}
}
