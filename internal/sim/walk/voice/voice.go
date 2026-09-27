package voice

// voiceError is the local rejection used by the walk harness for voice guards.
type voiceError string

func (e voiceError) Error() string { return string(e) }

var errVoiceBusy = voiceError("Mother Vell is speaking")

func guardMove(voiceBusy bool, npcReplies int, idleElapsed bool) error {
	if voiceBusy {
		return errVoiceBusy
	}
	if npcReplies == 0 && !idleElapsed {
		return voiceError("Talk to her first")
	}
	return nil
}
