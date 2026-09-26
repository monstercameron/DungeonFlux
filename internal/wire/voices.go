package wire

import (
	"context"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// elevenLabsDefaultVoices maps the canonical logical voices to ElevenLabs
// voice IDs. They match scripts/buildtime resolveVoiceID, so a live line and
// its canned fallback are spoken by the same voice.
func elevenLabsDefaultVoices() map[string]string {
	return map[string]string{
		"dm":          "21m00Tcm4TlvDq8ikWAM",
		"mother_vell": "EXAVITQu4vr4xnSDxMaL",
		"courier":     "pNInz6obpgDQGcFmaJgB",
	}
}

// canonicalVoice folds the engine's logical voice names onto the build-time
// names: content uses "voice_mother_vell", i18n uses "en-narrator", and the
// build-time renderer uses "mother_vell" and "dm".
func canonicalVoice(voice string) string {
	name := strings.ToLower(strings.TrimSpace(voice))
	name = strings.TrimPrefix(name, "voice_")
	if name == "narrator" || strings.HasSuffix(name, "-narrator") {
		return "dm"
	}
	return name
}

// newElevenLabsVoices resolves logical voices to ElevenLabs voice IDs. An
// environment override DF_ELEVENLABS_VOICE_<NAME> (for example
// DF_ELEVENLABS_VOICE_MOTHER_VELL) wins over the default, as in the build-time
// renderer.
func newElevenLabsVoices(lookup func(string) (string, bool)) func(string) string {
	defaults := elevenLabsDefaultVoices()
	return func(voice string) string {
		name := canonicalVoice(voice)
		if lookup != nil {
			if id, ok := lookup("DF_ELEVENLABS_VOICE_" + strings.ToUpper(name)); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
		if id, ok := defaults[name]; ok {
			return id
		}
		return voice
	}
}

// voiceMappedTTS translates the request's logical voice into a vendor voice
// ID before delegating. Unknown voices pass through unchanged, so a vendor ID
// set directly in content still works.
type voiceMappedTTS struct {
	inner   ports.TTS
	resolve func(string) string
}

func (t voiceMappedTTS) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	req.VoiceID = t.resolve(req.VoiceID)
	return t.inner.Stream(ctx, req, text)
}

// castVoice fills a spoken line's missing voice from the one-shot cast. The
// engine starts Mother Vell's and the courier's lines with a speaker but no
// voice, so they fell back to the narrator. A voice set by the engine wins.
func castVoice(cast []domain.NPC, effect domain.StartLine) domain.StartLine {
	if strings.TrimSpace(effect.Voice) != "" {
		return effect
	}
	npc := ""
	switch effect.Role {
	case vocab.RoleNPCReply, vocab.RoleNPCReveal, vocab.RoleNPCRefuse:
		npc = "mother_vell"
	case vocab.RoleStrangerLines:
		npc = "courier"
	default:
		return effect
	}
	for _, member := range cast {
		if string(member.ID) == npc && strings.TrimSpace(member.VoiceID) != "" {
			effect.Voice = member.VoiceID
			return effect
		}
	}
	return effect
}
