package vocab

// StateID identifies a top-level game state.
type StateID string

// MachineID identifies a state machine instance.
type MachineID string

// EventKind identifies an event handled by the engine.
type EventKind string

// EffectKind identifies work emitted by the engine.
type EffectKind string

// MoveID identifies a player move.
type MoveID string

// StatusID identifies a combat status.
type StatusID string

// Role identifies a model or narration role.
type Role string

// Slot identifies an asset slot.
type Slot string

// VendorName identifies an external or local vendor.
type VendorName string

// ReportKind identifies a client report.
type ReportKind string

// JobState identifies the state of an asynchronous job.
type JobState string

// SoundKind identifies a sound category.
type SoundKind string

// ErrKind identifies a normalized failure category.
type ErrKind string

// MsgRole identifies a chat message role.
type MsgRole string

// HostCmd identifies a host command.
type HostCmd string

// StreamKind identifies a client stream.
type StreamKind string

// RunMode identifies the mode of a game run.
type RunMode string

// AssetKind identifies an asset category.
type AssetKind string

// Phase identifies a game phase.
type Phase string

const (
	// StateLobby is the lobby state.
	StateLobby StateID = "lobby"
	// StateCreation is the character creation state.
	StateCreation StateID = "creation"
	// StateOpening is the opening state.
	StateOpening StateID = "opening"
	// StateExploration is the exploration state.
	StateExploration StateID = "exploration"
	// StateConversation is the conversation state.
	StateConversation StateID = "conversation"
	// StateCheck is the check state.
	StateCheck StateID = "check"
	// StateResolution is the resolution state.
	StateResolution StateID = "resolution"
	// StateHookEvent is the hook event state.
	StateHookEvent StateID = "hook_event"
	// StateCombat is the combat state.
	StateCombat StateID = "combat"
	// StateCliffhanger is the cliffhanger state.
	StateCliffhanger StateID = "cliffhanger"
	// StateEnd is the terminal state.
	StateEnd StateID = "end"
	// MachineRun identifies the run machine.
	MachineRun MachineID = "run"
	// MachineSession identifies the session machine.
	MachineSession MachineID = "session"
	// MachineCheck identifies the check machine.
	MachineCheck MachineID = "check"
	// MachineCombat identifies the combat machine.
	MachineCombat MachineID = "combat"
	// MachinePTT identifies the push-to-talk machine.
	MachinePTT MachineID = "ptt"
	// MachineSlot identifies the asset slot machine.
	MachineSlot MachineID = "slot"
)

const (
	// MoveReady readies a player.
	MoveReady MoveID = "ready"
	// MoveSpecies selects a species.
	MoveSpecies MoveID = "species"
	// MoveGender selects a gender.
	MoveGender MoveID = "gender"
	// MoveClass selects a character class.
	MoveClass MoveID = "class"
	// MoveRollHero rolls a hero.
	MoveRollHero MoveID = "roll_hero"
	// MoveTalkVell starts a conversation with Vell.
	MoveTalkVell MoveID = "talk_vell"
	// MovePersuade attempts persuasion.
	MovePersuade MoveID = "persuade"
	// MoveStepAway steps away from the conversation.
	MoveStepAway MoveID = "step_away"
	// MoveLeave leaves the scene.
	MoveLeave MoveID = "leave"
	// MoveAttack attacks a target.
	MoveAttack MoveID = "attack"
	// MoveMove moves a character.
	MoveMove MoveID = "move"
	// MoveEndTurn ends a combat turn.
	MoveEndTurn MoveID = "end_turn"
	// MoveDash spends the action to double movement and walk to a cell (R-D8).
	MoveDash MoveID = "dash"
	// StatusBloodied marks a bloodied combatant.
	StatusBloodied StatusID = "bloodied"
	// StatusDown marks a downed combatant.
	StatusDown StatusID = "down"
	// StatusDefeated marks a defeated combatant.
	StatusDefeated StatusID = "defeated"
	// StatusFled marks a fleeing combatant.
	StatusFled StatusID = "fled"
)

const (
	// RoleNPCReply requests an NPC reply.
	RoleNPCReply Role = "npc_reply"
	// RoleOpening requests opening narration.
	RoleOpening Role = "opening"
	// RoleNPCReveal requests an NPC reveal.
	RoleNPCReveal Role = "npc_reveal"
	// RoleNPCRefuse requests an NPC refusal.
	RoleNPCRefuse Role = "npc_refuse"
	// RoleCharacterFlavor requests character flavor.
	RoleCharacterFlavor Role = "character_flavor"
	// RoleInterpret requests interpretation.
	RoleInterpret Role = "interpret"
	// RoleStrangerLines requests stranger lines.
	RoleStrangerLines Role = "stranger_lines"
	// RoleCliffhanger requests cliffhanger narration.
	RoleCliffhanger Role = "cliffhanger"
	// RoleCombatOutcomes requests combat outcomes.
	RoleCombatOutcomes Role = "combat_outcomes"
	// SlotPortrait is the portrait asset slot.
	SlotPortrait Slot = "portrait"
	// SlotCliffhangerStill is the cliffhanger still asset slot.
	SlotCliffhangerStill Slot = "cliffhanger_still"
	// SlotCliffhangerClip is the cliffhanger clip asset slot.
	SlotCliffhangerClip Slot = "cliffhanger_clip"
	// SlotBattlefield is the battlefield asset slot.
	SlotBattlefield Slot = "battlefield"
	// VendorOpenAI identifies OpenAI.
	VendorOpenAI VendorName = "openai"
	// VendorGemini identifies Gemini.
	VendorGemini VendorName = "gemini"
	// VendorAnthropic identifies Anthropic.
	VendorAnthropic VendorName = "anthropic"
	// VendorElevenLabs identifies ElevenLabs.
	VendorElevenLabs VendorName = "elevenlabs"
	// VendorSegmind identifies Segmind.
	VendorSegmind VendorName = "segmind"
	// VendorEvoLink identifies EvoLink.
	VendorEvoLink VendorName = "evolink"
	// VendorFal identifies Fal.
	VendorFal VendorName = "fal"
	// VendorCerebras identifies Cerebras.
	VendorCerebras VendorName = "cerebras"
	// VendorLocal identifies the local vendor.
	VendorLocal VendorName = "local"
	// VendorKeyword identifies keyword matching.
	VendorKeyword VendorName = "keyword"
	// JobQueued means a job is queued.
	JobQueued JobState = "queued"
	// JobRunning means a job is running.
	JobRunning JobState = "running"
	// JobDone means a job completed.
	JobDone JobState = "done"
	// JobFailed means a job failed.
	JobFailed JobState = "failed"
	// SoundMusic identifies music.
	SoundMusic SoundKind = "music"
	// SoundAmbience identifies a continuous environmental bed.
	SoundAmbience SoundKind = "ambience"
	// SoundSFX identifies a sound effect.
	SoundSFX SoundKind = "sfx"
	// MsgSystem identifies a system message.
	MsgSystem MsgRole = "system"
	// MsgUser identifies a user message.
	MsgUser MsgRole = "user"
	// MsgAssistant identifies an assistant message.
	MsgAssistant MsgRole = "assistant"
	// RunLive identifies a live run.
	RunLive RunMode = "live"
	// RunStage identifies a staged run.
	RunStage RunMode = "stage"
	// RunRehearsal identifies a rehearsal run.
	RunRehearsal RunMode = "rehearsal"
)

const (
	// ReportPlaybackDone reports completed playback.
	ReportPlaybackDone ReportKind = "PLAYBACK_DONE"
	// ReportClipEnded reports an ended clip.
	ReportClipEnded ReportKind = "CLIP_ENDED"
	// ReportSplatReady reports a ready splat renderer.
	ReportSplatReady ReportKind = "SPLAT_READY"
	// ReportSplatFailed reports a failed splat renderer.
	ReportSplatFailed ReportKind = "SPLAT_FAILED"
	// ReportClientState reports client state.
	ReportClientState ReportKind = "CLIENT_STATE"
	// ReportClientScreenshot reports a client screenshot.
	ReportClientScreenshot ReportKind = "CLIENT_SCREENSHOT"
	// HostStart starts a run.
	HostStart HostCmd = "START"
	// HostPause pauses a run.
	HostPause HostCmd = "PAUSE"
	// HostResume resumes a run.
	HostResume HostCmd = "RESUME"
	// HostForceD20 forces a d20 result.
	HostForceD20 HostCmd = "FORCE_D20"
	// HostSafeMode enables safe mode.
	HostSafeMode HostCmd = "SAFE_MODE"
	// HostSkip skips the current step.
	HostSkip HostCmd = "SKIP"
	// HostReset resets the run.
	HostReset HostCmd = "RESET"
	// HostTimerAdd adds time to a timer.
	HostTimerAdd HostCmd = "TIMER_ADD"
	// HostTimersOff disables timers.
	HostTimersOff HostCmd = "TIMERS_OFF"
	// HostSplatOff disables the splat renderer.
	HostSplatOff HostCmd = "SPLAT_OFF"
	// StreamWatch identifies the watch stream.
	StreamWatch StreamKind = "watch"
	// StreamListen identifies the listen stream.
	StreamListen StreamKind = "listen"
	// StreamTalk identifies the talk stream.
	StreamTalk StreamKind = "talk"
	// AssetImage identifies an image asset.
	AssetImage AssetKind = "image"
	// AssetAudio identifies an audio asset.
	AssetAudio AssetKind = "audio"
	// AssetVideo identifies a video asset.
	AssetVideo AssetKind = "video"
	// AssetSplat identifies a splat asset.
	AssetSplat AssetKind = "splat"
	// AssetMusic identifies a music asset.
	AssetMusic AssetKind = "music"
	// AssetSFX identifies a sound effect asset.
	AssetSFX AssetKind = "sfx"
	// ErrTimeout identifies a timeout failure.
	ErrTimeout ErrKind = "timeout"
	// ErrRateLimited identifies a rate limit failure.
	ErrRateLimited ErrKind = "rate_limited"
	// ErrRefused identifies a refusal failure.
	ErrRefused ErrKind = "refused"
	// ErrUnavailable identifies an unavailable service failure.
	ErrUnavailable ErrKind = "unavailable"
	// ErrBadOutput identifies invalid output.
	ErrBadOutput ErrKind = "bad_output"
	// ErrCanceled identifies a canceled operation.
	ErrCanceled ErrKind = "canceled"
	// ErrAuth identifies an authentication failure.
	ErrAuth ErrKind = "auth"
)

const (
	// EventHostStart records a host start command.
	EventHostStart EventKind = "host_start"
	// EventHostReset records a host reset command.
	EventHostReset EventKind = "host_reset"
	// EventHostPause records a host pause command.
	EventHostPause EventKind = "host_pause"
	// EventHostResume records a host resume command.
	EventHostResume EventKind = "host_resume"
	// EventHostSkip records a host skip command.
	EventHostSkip EventKind = "host_skip"
	// EventHostForceD20 records a forced d20 command.
	EventHostForceD20 EventKind = "host_force_d20"
	// EventHostSafeMode records a safe mode command.
	EventHostSafeMode EventKind = "host_safe_mode"
	// EventHostTimerAdd records a timer add command.
	EventHostTimerAdd EventKind = "host_timer_add"
	// EventHostTimersOff records a timers off command.
	EventHostTimersOff EventKind = "host_timers_off"
	// EventHostSplatOff records a splat off command.
	EventHostSplatOff EventKind = "host_splat_off"
	// EventJoin records a player joining.
	EventJoin EventKind = "join"
	// EventAct records a player action.
	EventAct EventKind = "act"
	// EventSay records spoken text.
	EventSay EventKind = "say"
	// EventTalkStart records the start of talk.
	EventTalkStart EventKind = "talk_start"
	// EventTalkEnd records the end of talk.
	EventTalkEnd EventKind = "talk_end"
	// EventStreamClosed records a closed stream.
	EventStreamClosed EventKind = "stream_closed"
	// EventReport records a client report.
	EventReport EventKind = "report"
	// EventDebugReset records a debug reset.
	EventDebugReset EventKind = "debug_reset"
	// EventDebugGoto records a debug phase change.
	EventDebugGoto EventKind = "debug_goto"
	// EventDebugPatch records a debug patch.
	EventDebugPatch EventKind = "debug_patch"
	// EventDebugTimer records a debug timer command.
	EventDebugTimer EventKind = "debug_timer"
	// EventDebugForceDice records forced debug dice.
	EventDebugForceDice EventKind = "debug_force_dice"
	// EventTimerFired records a fired timer.
	EventTimerFired EventKind = "timer_fired"
	// EventTranscribed records transcribed speech.
	EventTranscribed EventKind = "transcribed"
	// EventSTTError records a speech-to-text error.
	EventSTTError EventKind = "stt_error"
	// EventInterpreted records interpreted input.
	EventInterpreted EventKind = "interpreted"
	// EventInterpretFailed records failed interpretation.
	EventInterpretFailed EventKind = "interpret_failed"
	// EventFlavorDone records completed character flavor.
	EventFlavorDone EventKind = "flavor_done"
	// EventFlavorFailed records failed character flavor.
	EventFlavorFailed EventKind = "flavor_failed"
	// EventLineFirstAudio records the first line audio.
	EventLineFirstAudio EventKind = "line_first_audio"
	// EventLineAudioFinal records final line audio.
	EventLineAudioFinal EventKind = "line_audio_final"
	// EventLineFailed records failed line audio.
	EventLineFailed EventKind = "line_failed"
	// EventNarrationDelta records a narration delta.
	EventNarrationDelta EventKind = "narration_delta"
	// EventAssetPartial records a partial asset.
	EventAssetPartial EventKind = "asset_partial"
	// EventAssetReady records a ready asset.
	EventAssetReady EventKind = "asset_ready"
	// EventAssetFailed records a failed asset.
	EventAssetFailed EventKind = "asset_failed"
	// EventPrerenderTextDone records completed text prerendering.
	EventPrerenderTextDone EventKind = "prerender_text_done"
	// EventPrerenderDone records completed prerendering.
	EventPrerenderDone EventKind = "prerender_done"
	// EventPrerenderFailed records failed prerendering.
	EventPrerenderFailed EventKind = "prerender_failed"
	// EventUtteranceFinal records a final utterance.
	EventUtteranceFinal EventKind = "utterance_final"
	// EventLineDone records completed line playback.
	EventLineDone EventKind = "line_done"
	// EventClipDone records completed clip playback.
	EventClipDone EventKind = "clip_done"
	// EventPCLocked records a locked player character.
	EventPCLocked EventKind = "pc_locked"
	// EventCombatStarted records combat start.
	EventCombatStarted EventKind = "combat_started"
	// EventTurnStarted records combat turn start.
	EventTurnStarted EventKind = "turn_started"
	// EventMoved records a combat move.
	EventMoved EventKind = "moved"
	// EventAttackMade records a combat attack.
	EventAttackMade EventKind = "attack_made"
	// EventDamageApplied records applied damage.
	EventDamageApplied EventKind = "damage_applied"
	// EventStatusApplied records an applied status.
	EventStatusApplied EventKind = "status_applied"
	// EventStatusRemoved records a removed status.
	EventStatusRemoved EventKind = "status_removed"
	// EventTurnEnded records combat turn end.
	EventTurnEnded EventKind = "turn_ended"
	// EventCombatEnded records combat end.
	EventCombatEnded EventKind = "combat_ended"
)
