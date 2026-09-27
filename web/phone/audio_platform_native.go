//go:build !js || !wasm

package phone

// PhoneAudio owns cue filtering and mute policy on native test builds.
type PhoneAudio struct {
	queue          *AudioQueue
	reducedMotion  bool
	vibrateEnabled bool
	localGate      localCueGate
}
