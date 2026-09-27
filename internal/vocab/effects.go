package vocab

const (
	// EffectStartTimer starts a timer.
	EffectStartTimer EffectKind = "start_timer"
	// EffectCancelTimer cancels a timer.
	EffectCancelTimer EffectKind = "cancel_timer"
	// EffectFreezeTimer freezes a timer.
	EffectFreezeTimer EffectKind = "freeze_timer"
	// EffectThawTimer thaws a timer.
	EffectThawTimer EffectKind = "thaw_timer"
	// EffectPauseAll pauses all work.
	EffectPauseAll EffectKind = "pause_all"
	// EffectResumeAll resumes all work.
	EffectResumeAll EffectKind = "resume_all"
	// EffectCancelScope cancels a scope.
	EffectCancelScope EffectKind = "cancel_scope"
	// EffectCancelKey cancels a keyed operation.
	EffectCancelKey EffectKind = "cancel_key"
	// EffectNewRun starts a new run.
	EffectNewRun EffectKind = "new_run"
	// EffectTalkStop stops talk playback.
	EffectTalkStop EffectKind = "talk_stop"
	// EffectSendAudioCancel cancels sent audio.
	EffectSendAudioCancel EffectKind = "send_audio_cancel"
	// EffectPlaySound routes a music, ambience, or sound-effect cue.
	EffectPlaySound EffectKind = "play_sound"
	// EffectTranscribe requests transcription.
	EffectTranscribe EffectKind = "transcribe"
	// EffectInterpret requests interpretation.
	EffectInterpret EffectKind = "interpret"
	// EffectCharacterFlavor requests character flavor.
	EffectCharacterFlavor EffectKind = "character_flavor"
	// EffectStartLine starts a voice line.
	EffectStartLine EffectKind = "start_line"
	// EffectReleaseLine releases a voice line.
	EffectReleaseLine EffectKind = "release_line"
	// EffectDropLine drops a voice line.
	EffectDropLine EffectKind = "drop_line"
	// EffectPlayCanned plays canned audio.
	EffectPlayCanned EffectKind = "play_canned"
	// EffectPrerenderText prerenders text.
	EffectPrerenderText EffectKind = "prerender_text"
	// EffectRenderLines renders voice lines.
	EffectRenderLines EffectKind = "render_lines"
	// EffectGenerateImage generates an image.
	EffectGenerateImage EffectKind = "generate_image"
	// EffectComposeStill composes a still image.
	EffectComposeStill EffectKind = "compose_still"
	// EffectGenerateClip generates a clip.
	EffectGenerateClip EffectKind = "generate_clip"
	// EffectGenerateBillboardLoops generates billboard loops.
	EffectGenerateBillboardLoops EffectKind = "generate_billboard_loops"
)
