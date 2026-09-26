package vocab

type StateID string
type MachineID string
type EventKind string
type EffectKind string
type MoveID string
type StatusID string
type Role string
type Slot string
type VendorName string
type ReportKind string
type JobState string
type SoundKind string
type ErrKind string
type MsgRole string
type HostCmd string
type StreamKind string
type RunMode string
type AssetKind string
type Phase string

const (
	StateLobby StateID = "lobby"; StateCreation StateID = "creation"; StateOpening StateID = "opening"; StateExploration StateID = "exploration"; StateConversation StateID = "conversation"; StateCheck StateID = "check"; StateResolution StateID = "resolution"; StateHookEvent StateID = "hook_event"; StateCombat StateID = "combat"; StateCliffhanger StateID = "cliffhanger"; StateEnd StateID = "end"
	MachineRun MachineID = "run"; MachineSession MachineID = "session"; MachineCheck MachineID = "check"; MachineCombat MachineID = "combat"; MachinePTT MachineID = "ptt"; MachineSlot MachineID = "slot"
)

const (
	MoveReady MoveID = "ready"; MoveSpecies MoveID = "species"; MoveGender MoveID = "gender"; MoveRollHero MoveID = "roll_hero"; MoveTalkVell MoveID = "talk_vell"; MovePersuade MoveID = "persuade"; MoveStepAway MoveID = "step_away"; MoveLeave MoveID = "leave"; MoveAttack MoveID = "attack"; MoveMove MoveID = "move"; MoveEndTurn MoveID = "end_turn"
	StatusBloodied StatusID = "bloodied"; StatusDown StatusID = "down"; StatusDefeated StatusID = "defeated"; StatusFled StatusID = "fled"
)

const (
	RoleNPCReply Role = "npc_reply"; RoleOpening Role = "opening"; RoleNPCReveal Role = "npc_reveal"; RoleNPCRefuse Role = "npc_refuse"; RoleCharacterFlavor Role = "character_flavor"; RoleInterpret Role = "interpret"; RoleStrangerLines Role = "stranger_lines"; RoleCliffhanger Role = "cliffhanger"; RoleCombatOutcomes Role = "combat_outcomes"
	SlotPortrait Slot = "portrait"; SlotCliffhangerStill Slot = "cliffhanger_still"; SlotCliffhangerClip Slot = "cliffhanger_clip"; SlotBattlefield Slot = "battlefield"
	VendorOpenAI VendorName = "openai"; VendorGemini VendorName = "gemini"; VendorAnthropic VendorName = "anthropic"; VendorElevenLabs VendorName = "elevenlabs"; VendorSegmind VendorName = "segmind"; VendorEvoLink VendorName = "evolink"; VendorFal VendorName = "fal"; VendorCerebras VendorName = "cerebras"; VendorLocal VendorName = "local"; VendorKeyword VendorName = "keyword"
	JobQueued JobState = "queued"; JobRunning JobState = "running"; JobDone JobState = "done"; JobFailed JobState = "failed"
	SoundMusic SoundKind = "music"; SoundSFX SoundKind = "sfx"
	MsgSystem MsgRole = "system"; MsgUser MsgRole = "user"; MsgAssistant MsgRole = "assistant"
	RunLive RunMode = "live"; RunStage RunMode = "stage"; RunRehearsal RunMode = "rehearsal"
)

const (
	ReportPlaybackDone ReportKind = "PLAYBACK_DONE"; ReportClipEnded ReportKind = "CLIP_ENDED"; ReportSplatReady ReportKind = "SPLAT_READY"; ReportSplatFailed ReportKind = "SPLAT_FAILED"; ReportClientState ReportKind = "CLIENT_STATE"; ReportClientScreenshot ReportKind = "CLIENT_SCREENSHOT"
	HostStart HostCmd = "START"; HostPause HostCmd = "PAUSE"; HostResume HostCmd = "RESUME"; HostForceD20 HostCmd = "FORCE_D20"; HostSafeMode HostCmd = "SAFE_MODE"; HostSkip HostCmd = "SKIP"; HostReset HostCmd = "RESET"; HostTimerAdd HostCmd = "TIMER_ADD"; HostTimersOff HostCmd = "TIMERS_OFF"; HostSplatOff HostCmd = "SPLAT_OFF"
	StreamWatch StreamKind = "watch"; StreamListen StreamKind = "listen"; StreamTalk StreamKind = "talk"
	AssetImage AssetKind = "image"; AssetAudio AssetKind = "audio"; AssetVideo AssetKind = "video"; AssetSplat AssetKind = "splat"; AssetMusic AssetKind = "music"; AssetSFX AssetKind = "sfx"
	ErrTimeout ErrKind = "timeout"; ErrRateLimited ErrKind = "rate_limited"; ErrRefused ErrKind = "refused"; ErrUnavailable ErrKind = "unavailable"; ErrBadOutput ErrKind = "bad_output"; ErrCanceled ErrKind = "canceled"; ErrAuth ErrKind = "auth"
)

const (
	EventHostStart EventKind = "host_start"; EventHostReset EventKind = "host_reset"; EventHostPause EventKind = "host_pause"; EventHostResume EventKind = "host_resume"; EventHostSkip EventKind = "host_skip"; EventHostForceD20 EventKind = "host_force_d20"; EventHostSafeMode EventKind = "host_safe_mode"; EventHostTimerAdd EventKind = "host_timer_add"; EventHostTimersOff EventKind = "host_timers_off"; EventHostSplatOff EventKind = "host_splat_off"; EventJoin EventKind = "join"; EventAct EventKind = "act"; EventSay EventKind = "say"; EventTalkStart EventKind = "talk_start"; EventTalkEnd EventKind = "talk_end"; EventStreamClosed EventKind = "stream_closed"; EventReport EventKind = "report"; EventDebugReset EventKind = "debug_reset"; EventDebugGoto EventKind = "debug_goto"; EventDebugPatch EventKind = "debug_patch"; EventDebugTimer EventKind = "debug_timer"; EventDebugForceDice EventKind = "debug_force_dice"; EventTimerFired EventKind = "timer_fired"; EventTranscribed EventKind = "transcribed"; EventSTTError EventKind = "stt_error"; EventInterpreted EventKind = "interpreted"; EventInterpretFailed EventKind = "interpret_failed"; EventFlavorDone EventKind = "flavor_done"; EventFlavorFailed EventKind = "flavor_failed"; EventLineFirstAudio EventKind = "line_first_audio"; EventLineAudioFinal EventKind = "line_audio_final"; EventLineFailed EventKind = "line_failed"; EventNarrationDelta EventKind = "narration_delta"; EventAssetPartial EventKind = "asset_partial"; EventAssetReady EventKind = "asset_ready"; EventAssetFailed EventKind = "asset_failed"; EventPrerenderTextDone EventKind = "prerender_text_done"; EventPrerenderDone EventKind = "prerender_done"; EventPrerenderFailed EventKind = "prerender_failed"; EventUtteranceFinal EventKind = "utterance_final"; EventLineDone EventKind = "line_done"; EventClipDone EventKind = "clip_done"; EventPCLocked EventKind = "pc_locked"; EventCombatStarted EventKind = "combat_started"; EventTurnStarted EventKind = "turn_started"; EventMoved EventKind = "moved"; EventAttackMade EventKind = "attack_made"; EventDamageApplied EventKind = "damage_applied"; EventStatusApplied EventKind = "status_applied"; EventStatusRemoved EventKind = "status_removed"; EventTurnEnded EventKind = "turn_ended"; EventCombatEnded EventKind = "combat_ended"
)

const (
	EffectStartTimer EffectKind = "start_timer"; EffectCancelTimer EffectKind = "cancel_timer"; EffectFreezeTimer EffectKind = "freeze_timer"; EffectThawTimer EffectKind = "thaw_timer"; EffectPauseAll EffectKind = "pause_all"; EffectResumeAll EffectKind = "resume_all"; EffectCancelScope EffectKind = "cancel_scope"; EffectCancelKey EffectKind = "cancel_key"; EffectNewRun EffectKind = "new_run"; EffectTalkStop EffectKind = "talk_stop"; EffectSendAudioCancel EffectKind = "send_audio_cancel"; EffectTranscribe EffectKind = "transcribe"; EffectInterpret EffectKind = "interpret"; EffectCharacterFlavor EffectKind = "character_flavor"; EffectStartLine EffectKind = "start_line"; EffectReleaseLine EffectKind = "release_line"; EffectDropLine EffectKind = "drop_line"; EffectPlayCanned EffectKind = "play_canned"; EffectPrerenderText EffectKind = "prerender_text"; EffectRenderLines EffectKind = "render_lines"; EffectGenerateImage EffectKind = "generate_image"; EffectComposeStill EffectKind = "compose_still"; EffectGenerateClip EffectKind = "generate_clip"; EffectGenerateBillboardLoops EffectKind = "generate_billboard_loops"
)
