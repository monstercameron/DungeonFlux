package vocab

import "testing"

func TestVocabularyValuesAreUnique(t *testing.T) {
	sets := map[string][]string{"states": {string(StateLobby), string(StateCreation), string(StateOpening), string(StateExploration), string(StateConversation), string(StateCheck), string(StateResolution), string(StateHookEvent), string(StateCombat), string(StateCliffhanger), string(StateEnd)}, "moves": {string(MoveReady), string(MoveSpecies), string(MoveGender), string(MoveRollHero), string(MoveTalkVell), string(MovePersuade), string(MoveStepAway), string(MoveLeave), string(MoveAttack), string(MoveMove), string(MoveEndTurn), string(MoveDash)}, "events": {string(EventHostStart), string(EventHostReset), string(EventHostPause), string(EventHostResume), string(EventHostSkip), string(EventHostForceD20), string(EventHostSafeMode), string(EventHostTimerAdd), string(EventHostTimersOff), string(EventHostSplatOff), string(EventJoin), string(EventAct), string(EventSay), string(EventTalkStart), string(EventTalkEnd), string(EventStreamClosed), string(EventReport), string(EventTimerFired), string(EventTranscribed), string(EventSTTError), string(EventInterpreted), string(EventInterpretFailed), string(EventLineDone), string(EventClipDone), string(EventCombatStarted), string(EventCombatEnded)}, "effects": {string(EffectStartTimer), string(EffectCancelTimer), string(EffectFreezeTimer), string(EffectThawTimer), string(EffectPauseAll), string(EffectResumeAll), string(EffectCancelScope), string(EffectCancelKey), string(EffectNewRun), string(EffectTalkStop), string(EffectSendAudioCancel), string(EffectTranscribe), string(EffectInterpret), string(EffectCharacterFlavor), string(EffectStartLine), string(EffectReleaseLine), string(EffectDropLine), string(EffectPlayCanned), string(EffectPrerenderText), string(EffectRenderLines), string(EffectGenerateImage), string(EffectComposeStill), string(EffectGenerateClip), string(EffectGenerateBillboardLoops)}}
	for name, values := range sets {
		seen := map[string]bool{}
		for _, value := range values {
			if value == "" || seen[value] {
				t.Fatalf("%s contains duplicate or empty value %q", name, value)
			}
			seen[value] = true
		}
	}
}
