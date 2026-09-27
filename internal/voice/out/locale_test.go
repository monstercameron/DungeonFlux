package out

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPCMExecutor_SelectsVoicePerLocale(t *testing.T) {
	cases := []struct {
		name      string
		room      string
		voices    map[string]string
		effect    string
		wantVoice string
	}{
		{"english default voice", "en", nil, "", "en-narrator"},
		{"spanish default voice", "es", nil, "", "es-narrator"},
		{"spanish override wins", "es", map[string]string{"es": "es-voice-1"}, "", "es-voice-1"},
		{"engine voice kept", "es", nil, "vell", "vell"},
		{"unknown room falls back", "fr", nil, "", "en-narrator"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			audio := &testAudio{}
			tts := &localeTTS{chunks: []ports.PCMChunk{{SampleRate: 16000, S16LE: []byte{1, 2}}}}
			executor := NewLineExecutor(tts, audio)
			room := tc.room
			executor.Room = func() string { return room }
			executor.Voices = tc.voices
			executor.Execute(context.Background(), domain.StartLine{
				UtteranceID: "line-locale", Role: vocab.RoleNPCReply, Voice: tc.effect, Input: "Hola.",
			}, domain.Scope{}, &testInbox{})
			if tts.request.VoiceID != tc.wantVoice {
				t.Fatalf("voice = %q, want %q", tts.request.VoiceID, tc.wantVoice)
			}
		})
	}
}

func TestCannedExecutor_PrefersLocalizedAsset(t *testing.T) {
	data := bytes.Repeat([]byte{7}, cannedChunkSize+2)
	assets := &localeAssets{data: map[string][]byte{
		"canned_npc_reply":    data,
		"canned_npc_reply_es": append([]byte(nil), data...),
	}, opened: nil}
	executor := NewCannedExecutor(assets, &testAudio{})
	executor.Room = func() string { return "es" }
	executor.PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-es", AssetID: "canned_npc_reply"}, domain.Scope{}, &testInbox{})
	if len(assets.opened) != 1 || assets.opened[0] != "canned_npc_reply_es" {
		t.Fatalf("opened = %v, want localized asset first", assets.opened)
	}
}

func TestCannedExecutor_FallsBackToBaseAsset(t *testing.T) {
	data := bytes.Repeat([]byte{7}, 4)
	assets := &localeAssets{data: map[string][]byte{"canned_npc_reply": data}}
	executor := NewCannedExecutor(assets, &testAudio{})
	executor.Room = func() string { return "es" }
	in := &testInbox{}
	executor.PlayCanned(context.Background(), domain.PlayCanned{UtteranceID: "c-fb", AssetID: "canned_npc_reply"}, domain.Scope{}, in)
	if len(assets.opened) != 2 || assets.opened[1] != "canned_npc_reply" {
		t.Fatalf("opened = %v, want base fallback", assets.opened)
	}
}

type localeTTS struct {
	request ports.TTSRequest
	chunks  []ports.PCMChunk
}

func (t *localeTTS) Stream(_ context.Context, request ports.TTSRequest, _ ports.TextStream) (ports.PCMStream, error) {
	t.request = request
	return &localePCM{chunks: t.chunks}, nil
}

type localePCM struct {
	chunks []ports.PCMChunk
	closed bool
}

func (s *localePCM) Recv() (ports.PCMChunk, error) {
	if len(s.chunks) == 0 {
		return ports.PCMChunk{}, io.EOF
	}
	next := s.chunks[0]
	s.chunks = s.chunks[1:]
	return next, nil
}

func (s *localePCM) Close() error {
	s.closed = true
	return nil
}

type localeAssets struct {
	data   map[string][]byte
	opened []domain.AssetID
}

func (a *localeAssets) Open(_ context.Context, id domain.AssetID) (io.ReadCloser, error) {
	a.opened = append(a.opened, id)
	data, ok := a.data[string(id)]
	if !ok {
		return nil, errors.New("locale asset is missing")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
