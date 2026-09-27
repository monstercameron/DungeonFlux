package wire

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestNewElevenLabsVoices_resolvesLogicalVoices(t *testing.T) {
	cases := []struct {
		name  string
		voice string
		env   map[string]string
		want  string
	}{
		{name: "dm", voice: "dm", want: "21m00Tcm4TlvDq8ikWAM"},
		{name: "content NPC voice", voice: "voice_mother_vell", want: "EXAVITQu4vr4xnSDxMaL"},
		{name: "content courier voice", voice: "voice_courier", want: "pNInz6obpgDQGcFmaJgB"},
		{name: "build-time NPC name", voice: "mother_vell", want: "EXAVITQu4vr4xnSDxMaL"},
		{name: "english narrator fallback", voice: "en-narrator", want: "21m00Tcm4TlvDq8ikWAM"},
		{name: "spanish narrator fallback", voice: "es-narrator", want: "21m00Tcm4TlvDq8ikWAM"},
		{name: "case and space", voice: " Voice_Mother_Vell ", want: "EXAVITQu4vr4xnSDxMaL"},
		{name: "env override", voice: "voice_mother_vell", env: map[string]string{"DF_ELEVENLABS_VOICE_MOTHER_VELL": "customVellID"}, want: "customVellID"},
		{name: "blank override ignored", voice: "dm", env: map[string]string{"DF_ELEVENLABS_VOICE_DM": "  "}, want: "21m00Tcm4TlvDq8ikWAM"},
		{name: "vendor id passes through", voice: "AZnzlk1XvdvUeBnXmlld", want: "AZnzlk1XvdvUeBnXmlld"},
		{name: "empty stays empty", voice: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				value, ok := tc.env[key]
				return value, ok
			}
			if got := newElevenLabsVoices(lookup)(tc.voice); got != tc.want {
				t.Fatalf("resolve(%q) = %q, want %q", tc.voice, got, tc.want)
			}
		})
	}
}

func TestNewElevenLabsVoices_nilLookupUsesDefaults(t *testing.T) {
	if got := newElevenLabsVoices(nil)("dm"); got != "21m00Tcm4TlvDq8ikWAM" {
		t.Fatalf("resolve(dm) = %q, want the default DM voice", got)
	}
}

type capturingTTS struct{ req ports.TTSRequest }

func (c *capturingTTS) Stream(_ context.Context, req ports.TTSRequest, _ ports.TextStream) (ports.PCMStream, error) {
	c.req = req
	return nil, nil
}

func TestVoiceMappedTTS_sendsVendorVoiceID(t *testing.T) {
	inner := &capturingTTS{}
	tts := voiceMappedTTS{inner: inner, resolve: newElevenLabsVoices(nil)}
	if _, err := tts.Stream(context.Background(), ports.TTSRequest{VoiceID: "voice_mother_vell", SampleRate: 24000}, nil); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if inner.req.VoiceID != "EXAVITQu4vr4xnSDxMaL" {
		t.Fatalf("inner VoiceID = %q, want the Mother Vell ElevenLabs voice", inner.req.VoiceID)
	}
	if inner.req.SampleRate != 24000 {
		t.Fatalf("inner SampleRate = %d, want the request's other fields kept", inner.req.SampleRate)
	}
}

func TestLiveTTS_elevenLabsMapsVoices(t *testing.T) {
	cfg := config.Config{Adapters: map[string]config.AdapterConfig{"tts": {Vendor: "elevenlabs", Mode: "live", APIKey: "test"}}}
	tts, err := liveTTS(cfg, nil)
	if err != nil {
		t.Fatalf("liveTTS: %v", err)
	}
	if _, ok := tts.(voiceMappedTTS); !ok {
		t.Fatalf("liveTTS(elevenlabs) = %T, want voiceMappedTTS", tts)
	}
}

func TestCastVoice_fillsCharacterVoices(t *testing.T) {
	cast := []domain.NPC{{ID: "mother_vell", VoiceID: "voice_mother_vell"}, {ID: "courier", VoiceID: "voice_courier"}}
	cases := []struct {
		name  string
		role  vocab.Role
		voice string
		want  string
	}{
		{name: "vell reply", role: vocab.RoleNPCReply, want: "voice_mother_vell"},
		{name: "vell reveal", role: vocab.RoleNPCReveal, want: "voice_mother_vell"},
		{name: "vell refuse", role: vocab.RoleNPCRefuse, want: "voice_mother_vell"},
		{name: "courier line", role: vocab.RoleStrangerLines, want: "voice_courier"},
		{name: "engine voice wins", role: vocab.RoleNPCReply, voice: "dm", want: "dm"},
		{name: "narration untouched", role: vocab.RoleOpening, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := castVoice(cast, domain.StartLine{Role: tc.role, Voice: tc.voice})
			if got.Voice != tc.want {
				t.Fatalf("voice = %q, want %q", got.Voice, tc.want)
			}
		})
	}
	if got := castVoice(nil, domain.StartLine{Role: vocab.RoleNPCReply}); got.Voice != "" {
		t.Fatalf("empty cast voice = %q, want none", got.Voice)
	}
}

func TestCastVoice_defaultCastResolvesToDistinctVoices(t *testing.T) {
	resolve := newElevenLabsVoices(nil)
	cast := content.DefaultOneShot().NPCs
	vell := resolve(castVoice(cast, domain.StartLine{Role: vocab.RoleNPCReply}).Voice)
	courier := resolve(castVoice(cast, domain.StartLine{Role: vocab.RoleStrangerLines}).Voice)
	narrator := resolve("dm")
	if vell != "EXAVITQu4vr4xnSDxMaL" || courier != "pNInz6obpgDQGcFmaJgB" || vell == narrator {
		t.Fatalf("vell = %q, courier = %q, narrator = %q; want three distinct ElevenLabs voices", vell, courier, narrator)
	}
}
