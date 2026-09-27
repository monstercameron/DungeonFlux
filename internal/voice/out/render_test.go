package out

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestRenderLinesExecutor_ExecutePostsAssetsInOrder(t *testing.T) {
	tts := &fakes.FakeTTS{Script: []fakes.TTSResult{
		{Chunks: []ports.PCMChunk{{SampleRate: 24000, S16LE: []byte{1, 2}}, {S16LE: []byte{3, 4}}}},
		{Chunks: []ports.PCMChunk{{SampleRate: 12000, S16LE: []byte{5, 6, 7, 8}}}},
	}}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{
		{Asset: domain.Asset{ID: "a1"}}, {Asset: domain.Asset{ID: "a2"}},
	}}
	in := &fakes.FakeInbox{}
	e := NewRenderLinesExecutor(tts, writer)
	e.Execute(t.Context(), domain.RenderLines{Set: "stranger", Texts: []string{"first", "second"}, Voices: []string{"v1", "v2"}}, domain.Scope{Key: "run"}, in)

	if len(writer.Calls) != 2 || string(writer.Calls[0].Data) != string([]byte{1, 2, 3, 4}) || writer.Calls[1].Meta.DurationMS != 0 {
		t.Fatalf("writes=%+v", writer.Calls)
	}
	if len(in.Calls) != 1 {
		t.Fatalf("posts=%d", len(in.Calls))
	}
	event, ok := in.Calls[0].Envelope.Event.(domain.PrerenderDone)
	if !ok || event.Set != "stranger" || len(event.Assets) != 2 || event.Assets[1].ID != "a2" {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
	if got := tts.Calls[0].Request.SampleRate; got != renderSampleRate {
		t.Fatalf("sample rate=%d", got)
	}
}

func TestRenderLinesExecutor_ExecutePostsFailureForValidationOrTTS(t *testing.T) {
	tests := []struct {
		name   string
		effect domain.RenderLines
		script []fakes.TTSResult
	}{
		{name: "validation", effect: domain.RenderLines{Set: "set", Texts: []string{"text"}, Voices: []string{""}}},
		{name: "tts error", effect: domain.RenderLines{Set: "set", Texts: []string{"text"}, Voices: []string{"voice"}}, script: []fakes.TTSResult{{Err: errors.New("down")}}},
		{name: "empty pcm", effect: domain.RenderLines{Set: "set", Texts: []string{"text"}, Voices: []string{"voice"}}, script: []fakes.TTSResult{{Chunks: []ports.PCMChunk{}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &fakes.FakeInbox{}
			NewRenderLinesExecutor(&fakes.FakeTTS{Script: tt.script}, &fakes.FakeAssetWriter{}).Execute(t.Context(), tt.effect, domain.Scope{}, in)
			if len(in.Calls) != 1 {
				t.Fatalf("posts=%d", len(in.Calls))
			}
			if _, ok := in.Calls[0].Envelope.Event.(domain.PrerenderFailed); !ok {
				t.Fatalf("event=%T", in.Calls[0].Envelope.Event)
			}
		})
	}
}

func TestRenderLinesExecutor_ExecuteCancellationDoesNotPost(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	in := &fakes.FakeInbox{}
	NewRenderLinesExecutor(&fakes.FakeTTS{}, &fakes.FakeAssetWriter{}).Execute(ctx, domain.RenderLines{Set: "set", Texts: []string{"text"}, Voices: []string{"voice"}}, domain.Scope{}, in)
	if len(in.Calls) != 0 {
		t.Fatalf("posts=%d", len(in.Calls))
	}
}

func TestSingleText_EmitsOnce(t *testing.T) {
	stream := &singleText{text: "line"}
	text, err := stream.Recv()
	if text != "line" || err != nil {
		t.Fatalf("first=%q %v", text, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second=%v", err)
	}
}

func TestReadPCM_ReturnsStreamError(t *testing.T) {
	_, _, err := readPCM(&errorPCMStream{})
	if err == nil {
		t.Fatal("expected stream error")
	}
}

type errorPCMStream struct{}

func (*errorPCMStream) Recv() (ports.PCMChunk, error) { return ports.PCMChunk{}, errors.New("broken") }
func (*errorPCMStream) Close() error                  { return nil }

var _ ports.TTS = (*fakes.FakeTTS)(nil)
var _ ports.AssetWriter = (*fakes.FakeAssetWriter)(nil)
var _ ports.Inbox = (*fakes.FakeInbox)(nil)
var _ vocab.AssetKind = vocab.AssetAudio
