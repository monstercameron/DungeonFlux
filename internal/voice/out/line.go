package out

import (
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// LineExecutor is the live StartLine executor exposed to runtime wiring.
//
// PCMExecutor contains the transport implementation shared with canned
// playback. Keeping this name at the voice-line boundary lets composition
// register live lines without coupling it to the PCM implementation detail.
type LineExecutor = PCMExecutor

// NewLineExecutor creates a live line executor that streams TTS PCM to audio.
func NewLineExecutor(tts ports.TTS, audio ports.AudioOut) *LineExecutor {
	return NewPCMExecutor(tts, audio)
}
