package wire

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func isFakeMode(cfg config.Config) bool {
	for _, adapter := range cfg.Adapters {
		if adapter.Mode == "fake" {
			return true
		}
	}
	return false
}

// fakeLLM is the deterministic model used by config/fake.json. Its output is
// deliberately valid for every structured role used by the demo.
type fakeLLM struct{}

func newFakeLLM() ports.LLM { return fakeLLM{} }

func (fakeLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	text := "The rain hammers the Drowned Lantern. The bell tower waits beyond the river."
	if req.Meta.Role == vocab.RoleNPCReply {
		text = "Mother Vell studies you, amused. Ask your question plainly."
	}
	return &fakeTextStream{ctx: ctx, text: text}, nil
}

func (fakeLLM) JSON(ctx context.Context, req ports.TextRequest, _ ports.Schema) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var values map[string]any
	switch req.Meta.Role {
	case vocab.RoleCharacterFlavor:
		values = map[string]any{"name": "Asha", "look": "A rain-dark cloak and bright eyes", "hook": "Find the vanished lamplighter"}
	case vocab.RoleInterpret:
		values = map[string]any{"clean_text": "I persuade her", "kind": "MOVE", "move_id": string(vocab.MovePersuade)}
	case vocab.RoleStrangerLines:
		values = map[string]any{"found": "The letter followed me from the river.", "not_found": "It followed me from the river."}
	case vocab.RoleCliffhanger:
		values = map[string]any{"found_via_npc": "The bell tolls midnight, and someone knows your names.", "found_via_stranger": "The letter opens as the bell tolls midnight."}
	case vocab.RoleCombatOutcomes:
		values = map[string]any{"slain_by_seat1": "The drowned thrall falls beneath your blade.", "slain_by_seat2": "The drowned thrall sinks back into the river-dark.", "fled": "The bell tolls, and the thrall flees toward the tower."}
	default:
		return nil, errors.New("wire: fake LLM has no structured response")
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type fakeTextStream struct {
	ctx  context.Context
	text string
	done bool
}

func (s *fakeTextStream) Recv() (string, error) {
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	if s.done {
		return "", io.EOF
	}
	s.done = true
	return s.text, nil
}

func (s *fakeTextStream) Close() error {
	s.done = true
	return nil
}

type fakeImage struct{ png []byte }

func newFakeImage() ports.ImageGen { return &fakeImage{png: fakePNG()} }

func (f *fakeImage) Generate(ctx context.Context, _ ports.ImageRequest) (ports.ImageStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &fakeImageStream{ctx: ctx, png: append([]byte(nil), f.png...)}, nil
}

type fakeImageStream struct {
	ctx   context.Context
	png   []byte
	index int
}

func (s *fakeImageStream) Recv() (ports.ImageEvent, error) {
	if err := s.ctx.Err(); err != nil {
		return ports.ImageEvent{}, err
	}
	if s.index > 1 {
		return ports.ImageEvent{}, io.EOF
	}
	event := ports.ImageEvent{PNG: append([]byte(nil), s.png...), Partial: s.index == 0, Index: s.index}
	s.index++
	return event, nil
}

func (s *fakeImageStream) Close() error { return nil }

type fakeVideo struct{}

func newFakeVideo() ports.VideoGen { return fakeVideo{} }

func (fakeVideo) Submit(ctx context.Context, _ ports.VideoRequest) (ports.VideoJob, error) {
	if err := ctx.Err(); err != nil {
		return ports.VideoJob{}, err
	}
	return ports.VideoJob{Vendor: "fake", ID: "fake-video"}, nil
}

func (fakeVideo) Poll(ctx context.Context, _ ports.VideoJob) (ports.VideoStatus, error) {
	if err := ctx.Err(); err != nil {
		return ports.VideoStatus{}, err
	}
	return ports.VideoStatus{State: vocab.JobDone, URL: "fake://video"}, nil
}

func (fakeVideo) Download(ctx context.Context, _ string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []byte("fake-video"), nil
}

type fakeSTT struct{}

func (fakeSTT) Transcribe(ctx context.Context, _ ports.STTRequest) (ports.Transcript, error) {
	if err := ctx.Err(); err != nil {
		return ports.Transcript{}, err
	}
	return ports.Transcript{Text: "persuade"}, nil
}

type fakeTTS struct{}

func (fakeTTS) Stream(ctx context.Context, _ ports.TTSRequest, _ ports.TextStream) (ports.PCMStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &fakePCMStream{ctx: ctx, pcm: fakePCM()}, nil
}

type fakePCMStream struct {
	ctx  context.Context
	pcm  []byte
	done bool
	wait bool
}

func (s *fakePCMStream) Recv() (ports.PCMChunk, error) {
	if err := s.ctx.Err(); err != nil {
		return ports.PCMChunk{}, err
	}
	if s.done {
		return ports.PCMChunk{}, io.EOF
	}
	if s.wait {
		timer := time.NewTimer(1200 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-s.ctx.Done():
			return ports.PCMChunk{}, s.ctx.Err()
		}
		s.done = true
		return ports.PCMChunk{}, io.EOF
	}
	s.wait = true
	return ports.PCMChunk{SampleRate: 24000, S16LE: append([]byte(nil), s.pcm...)}, nil
}

func (s *fakePCMStream) Close() error { return nil }

func fakePCM() []byte { return make([]byte, 24000*2*5/4) }

func fakePNG() []byte {
	var buffer bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			picture.Set(x, y, color.RGBA{R: 38, G: 52, B: 60, A: 255})
		}
	}
	_ = png.Encode(&buffer, picture)
	return buffer.Bytes()
}
