package i18n

import "sort"

// RequiredKeys lists every key the game and clients may render. The guard
// test fails when a catalog misses one of these or carries a key outside it.
func RequiredKeys() []string {
	keys := make([]string, 0, len(EnglishEntries()))
	for key := range EnglishEntries() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// CannedKey maps a canned-line asset ID to its catalog key.
func CannedKey(assetID string) string {
	return "canned." + assetID
}

// MoveKey maps a move ID (for example "talk_vell") to its catalog key.
func MoveKey(moveID string) string {
	return "move." + moveID
}

// ReasonKey maps a reason code (for example "SPEAKING") to its catalog key.
func ReasonKey(code string) string {
	return "reason." + code
}

// ScreenKeys groups UI keys by the screen that renders them, so the guard
// test can attribute hard-coded text to a screen.
func ScreenKeys() map[string][]string {
	return map[string][]string{
		"phone": {
			"move.ready", "move.species", "move.gender", "move.class", "move.roll_hero", "move.rename",
			"move.talk_vell", "move.persuade", "move.step_away", "move.leave",
			"move.attack", "move.move", "move.end_turn",
			"reason.WAITING_FOR_PLAYER", "reason.CONVERSATION_DONE",
			"reason.CONVERSATION_REFUSED", "reason.TALK_FIRST", "reason.SPEAKING",
			"reason.NO_PATH", "reason.NO_MOVEMENT", "reason.DOWN",
			"reason.BUILDING_HERO", "reason.MISSING_CHOICE", "reason.MISSING_CLASS", "reason.NOT_YOUR_TURN",
			"reason.NO_SLOT", "reason.OUT_OF_RANGE", "reason.INCAPACITATED",
			"ui.join.title", "ui.join.room_code", "ui.join.join", "ui.join.retry",
			"ui.join.room_required", "ui.join.failed", "ui.join.waiting",
			"ui.ptt.hold", "ui.ptt.recording", "ui.ptt.transcribe", "ui.ptt.failed",
			"ui.typed.hint", "ui.typed.send", "ui.dice.roll", "ui.dice.success",
			"ui.dice.failure", "ui.combat.your_turn", "ui.combat.waiting",
			"ui.sheet.hp", "ui.sheet.ac", "ui.shell.join", "ui.errors.unavail",
			"create.title", "create.hint", "create.species", "create.gender",
			"moves.title",
			"dice.title", "dice.roll", "dice.rolling", "dice.result", "dice.label",
			"dice.button",
			"typed.label", "typed.send", "typed.sent",
			"sheet.yours", "sheet.hp_short", "sheet.hp_none", "sheet.no_conditions",
			"sheet.conditions",
			"ptt.start", "ptt.stop", "ptt.ready", "ptt.norecord", "ptt.mic_canceled",
			"ptt.mic_unavail", "ptt.finishing", "ptt.unavail",
			"phone.error_title", "phone.client_unavail",
			"phone.check.difficulty", "phone.class.title", "phone.combat.move", "phone.combat.dash",
			"phone.combat.enemy", "phone.combat.enemy_glyph", "phone.combat.unavailable", "phone.combat.vitals",
			"phone.combat.target", "phone.hp", "phone.create.controls", "phone.create.empty", "phone.create.glyph",
			"phone.create.rename_action", "phone.create.rename_title", "phone.create.rename_placeholder",
			"phone.create.rename_save", "phone.create.rename_cancel", "phone.create.rename_error_empty",
			"phone.create.rename_error_too_long",
			"phone.explore.empty", "phone.brand.dungeon", "phone.brand.flux", "phone.sheet.saves",
			"phone.sheet.skills", "phone.sheet.attack", "phone.talk.waiting", "phone.talk.send",
			"phone.talk.send_glyph", "phone.waiting.empty", "phone.waiting.glyph", "phone.waiting.title",
			"phone.waiting.at_table",
			"tabs.character", "tabs.journal", "tabs.play", "tabs.map", "tabs.menu",
			"turn.yours", "turn.waiting_named", "turn.waiting",
			"journal.title", "journal.subtitle", "journal.empty", "journal.dm_speaker",
			"map.title", "map.subtitle", "map.empty", "map.combat_link", "map.location",
			"inventory.title", "inventory.empty", "inventory.worn",
			"menu.title", "menu.language", "menu.language_en", "menu.language_es",
			"menu.sound", "menu.sound_on", "menu.sound_off", "menu.help", "menu.help_body",
			"menu.leave", "menu.leave_confirm_title", "menu.leave_confirm_body",
			"menu.leave_confirm_yes", "menu.leave_confirm_cancel",
			"phone.audio.enable", "phone.audio.mute", "phone.audio.unmute",
			"sheet.tab_actions", "sheet.tab_inventory", "sheet.tab_spells", "sheet.tab_info", "sheet.no_spells",
		},
		"dm": {
			"canned.canned_opening", "canned.canned_npc_reply", "canned.canned_npc_reveal",
			"canned.canned_npc_refuse", "canned.canned_stranger_found",
			"canned.canned_stranger_relocated", "canned.canned_cliffhanger_npc",
			"canned.canned_cliffhanger_stranger", "canned.canned_slain_by_seat1",
			"canned.canned_slain_by_seat2", "canned.canned_fled",
			"canned.canned_nudge_exploration", "canned.canned_nudge_conversation",
			"ui.dm.narration", "ui.dm.callout", "ui.dm.dice_offer", "ui.dm.dice_rolled",
			"ui.dm.round", "ui.dm.lobby", "ui.dm.paused", "ui.dm.combat", "ui.dm.end",
			"ui.sheet.hp", "ui.sheet.ac",
			"dm.lobby_title", "dm.lobby_intro", "dm.room_code", "dm.seats_label",
			"dm.listen", "dm.waiting_join", "dm.joined", "dm.ready", "dm.seat",
			"dm.audio_waiting", "dm.audio_unlock", "dm.error_title", "dm.eyebrow_live",
			"dm.eyebrow_end", "dm.end_rules", "dm.dice_label", "dm.kind_check",
			"dm.kind_attack", "dm.vs_dc", "dm.crit", "dm.timer_paused", "dm.timer_label",
			"dm.scene_label", "dm.scene_heroes", "dm.combat_label", "dm.dice_aria",
			"dm.qr_alt", "dm.clip_fallback", "dm.clip_still", "dm.clip_label",
			"dm.end_title", "dm.end_subtitle", "dm.end_header", "dm.end_hook", "dm.end_party", "dm.end_next",
			"dm.cliffhanger_continued",
			"canned.canned_cliffhanger_vell", "canned.canned_combat_slain_seat1", "canned.canned_combat_slain_seat2",
			"dm.callout.steering", "dm.callout.prompt", "dm.combat.title", "dm.combat.enemy", "dm.combat.round",
			"dm.glyph.chevron", "dm.glyph.star", "dm.glyph.objective", "dm.glyph.minimap", "dm.glyph.music",
			"dm.brand", "dm.hp_unavailable", "dm.speaker_alt", "dm.create.title", "dm.create.hint",
			"dm.create.stats", "dm.create.phone_controls", "dm.create.step_gender", "dm.create.step_race",
			"dm.create.step_class", "dm.create.generate", "dm.create.choices", "dm.create.glyph",
			"dm.lobby.tagline", "dm.lobby.quote", "dm.lobby.scan", "dm.lobby.qr", "dm.lobby.qr_alt",
			"dm.lobby.party_footer", "dm.scene.current",
		},
		"host": {
			"ui.host.start", "ui.host.pause", "ui.host.resume", "ui.host.skip",
			"ui.host.reset", "ui.host.force1", "ui.host.force20", "ui.host.safe_mode",
			"ui.host.timers", "ui.host.splat", "ui.host.room_lang", "ui.host.waiting",
			"ui.host.language", "ui.dm.lobby", "ui.dm.paused",
			"host.title", "host.run", "host.assets", "host.log", "host.no_snapshot",
			"host.no_assets", "host.no_logs", "host.mode", "host.next_d20",
			"host.combat_cap", "host.sending", "host.accepted", "host.rejected",
			"host.error", "host.connecting", "host.ready",
			"host.tester_links_title", "host.copy_link", "host.kicker",
			"host.controls", "host.controls_hint", "host.stage_tools",
			"host.stage_tools_dice", "host.stage_tools_flags", "host.links_hint",
			"host.link.dm", "host.link.phone", "host.link.host",
			"host.token_reveal", "host.token_hidden",
			"host.reset_confirm_title", "host.reset_confirm_body", "host.reset_confirm_cancel",
			"host.status.phase_line", "host.status.spotlight_line", "host.status.turn_timer_line",
			"host.status.phase", "host.status.turn", "host.status.seats", "host.status.seats_value",
			"host.status.timers_on", "host.status.timers_off", "host.status.ok",
			"host.status.failures", "host.status.turn_none",
		},
		"shell": {
			"shell.join_title", "shell.join_scan", "shell.join_table", "shell.joining",
			"shell.seat_ready", "shell.room_ph", "shell.brand", "shell.redirect",
			"about.title", "about.rules",
			"ui.join.title", "ui.join.room_code", "ui.join.join", "ui.join.retry",
			"ui.join.room_required", "ui.join.failed", "ui.join.waiting",
		},
	}
}
