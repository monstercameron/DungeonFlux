package host

import (
	"strconv"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
)

// catalog is the committed client catalog shared by every host screen.
var catalog = i18n.Default()

// T renders a catalog key for a host locale with argument substitution and
// English fallback. Every host screen renders static text through T.
func T(locale, key string, args map[string]string) string {
	return catalog.T(locale, key, args, 0, "")
}

// localeOrDefault normalizes a render locale for host views.
func localeOrDefault(locale string) string {
	if locale == "" {
		return i18n.DefaultLocale
	}
	return i18n.Settle(locale, i18n.DefaultLocale)
}

// hostActionKey maps a host action to its catalog label key.
func hostActionKey(action hostAction) string {
	switch action.Command {
	case df.HostCommandKind_HOST_COMMAND_KIND_START:
		return "ui.host.start"
	case df.HostCommandKind_HOST_COMMAND_KIND_PAUSE:
		return "ui.host.pause"
	case df.HostCommandKind_HOST_COMMAND_KIND_RESUME:
		return "ui.host.resume"
	case df.HostCommandKind_HOST_COMMAND_KIND_SKIP:
		return "ui.host.skip"
	case df.HostCommandKind_HOST_COMMAND_KIND_RESET:
		return "ui.host.reset"
	case df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20:
		if action.D20 == 1 {
			return "ui.host.force1"
		}
		return "ui.host.force20"
	case df.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE:
		return "ui.host.safe_mode"
	case df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF:
		return "ui.host.timers"
	case df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF:
		return "ui.host.splat"
	default:
		return ""
	}
}

// HostActionLabel returns the localized button label for a host action.
func HostActionLabel(locale string, action hostAction) string {
	if key := hostActionKey(action); key != "" {
		return T(locale, key, nil)
	}
	return action.Label
}

// HostTitle returns the localized host page heading.
func HostTitle(locale string) string { return T(locale, "host.title", nil) }

// RunSection returns the localized run-status section heading.
func RunSection(locale string) string { return T(locale, "host.run", nil) }

// AssetsSection returns the localized asset-slots section heading.
func AssetsSection(locale string) string { return T(locale, "host.assets", nil) }

// LogSection returns the localized log-tail section heading.
func LogSection(locale string) string { return T(locale, "host.log", nil) }

// NoSnapshot returns the localized empty run-status line.
func NoSnapshot(locale string) string { return T(locale, "host.no_snapshot", nil) }

// NoAssets returns the localized empty assets line.
func NoAssets(locale string) string { return T(locale, "host.no_assets", nil) }

// NoLogs returns the localized empty log line.
func NoLogs(locale string) string { return T(locale, "host.no_logs", nil) }

// ModeLine returns the localized run-mode line.
func ModeLine(locale, mode string) string {
	return T(locale, "host.mode", map[string]string{"mode": mode})
}

// NextD20Line returns the localized next-d20 line.
func NextD20Line(locale string, d20 int32) string {
	return T(locale, "host.next_d20", map[string]string{"n": strconv.Itoa(int(d20))})
}

// CombatCapLine returns the localized combat-cap line.
func CombatCapLine(locale string, ms int64) string {
	return T(locale, "host.combat_cap", map[string]string{"ms": strconv.FormatInt(ms, 10)})
}

// SendingLine returns the localized in-flight command status.
func SendingLine(locale, label string) string {
	return T(locale, "host.sending", map[string]string{"label": label})
}

// AcceptedLine returns the localized accepted-command status.
func AcceptedLine(locale string) string { return T(locale, "host.accepted", nil) }

// RejectedLine returns the localized rejected-command status.
func RejectedLine(locale, reason string) string {
	return T(locale, "host.rejected", map[string]string{"reason": reason})
}

// ErrorLine returns the localized command-error status.
func ErrorLine(locale, reason string) string {
	return T(locale, "host.error", map[string]string{"reason": reason})
}

// ConnectingLine returns the localized connecting status.
func ConnectingLine(locale string) string { return T(locale, "host.connecting", nil) }

// ReadyLine returns the localized ready status.
func ReadyLine(locale string) string { return T(locale, "host.ready", nil) }

// RoomLocaleLabel returns the localized room-language selector caption.
func RoomLocaleLabel(locale string) string { return T(locale, "ui.host.room_lang", nil) }

// TesterLinksTitle labels the copyable links used during a rehearsal.
func TesterLinksTitle(locale string) string { return T(locale, "host.tester_links_title", nil) }

// CopyLinkLabel labels a copy action.
func CopyLinkLabel(locale string) string { return T(locale, "host.copy_link", nil) }

// HostKicker returns the small "CONTROL ROOM" label above the host title.
func HostKicker(locale string) string { return T(locale, "host.kicker", nil) }

// ControlsHeading returns the localized "Run controls" section heading.
func ControlsHeading(locale string) string { return T(locale, "host.controls", nil) }

// ControlsHint returns the localized hint under the run-controls heading.
func ControlsHint(locale string) string { return T(locale, "host.controls_hint", nil) }

// StageToolsHeading returns the localized "Stage tools" section heading.
func StageToolsHeading(locale string) string { return T(locale, "host.stage_tools", nil) }

// StageToolsDice labels the force-d20 sub-group inside Stage tools.
func StageToolsDice(locale string) string { return T(locale, "host.stage_tools_dice", nil) }

// StageToolsFlags labels the toggle sub-group inside Stage tools.
func StageToolsFlags(locale string) string { return T(locale, "host.stage_tools_flags", nil) }

// LinksHint returns the localized hint under the tester-links heading.
func LinksHint(locale string) string { return T(locale, "host.links_hint", nil) }

// LinkLabelDM, LinkLabelPhone, and LinkLabelHost name each copyable link.
func LinkLabelDM(locale string) string    { return T(locale, "host.link.dm", nil) }
func LinkLabelPhone(locale string) string { return T(locale, "host.link.phone", nil) }
func LinkLabelHost(locale string) string  { return T(locale, "host.link.host", nil) }

// TokenRevealLabel labels the hold-to-reveal control on a masked link.
func TokenRevealLabel(locale string) string { return T(locale, "host.token_reveal", nil) }

// TokenHiddenTitle explains why a link's token is masked.
func TokenHiddenTitle(locale string) string { return T(locale, "host.token_hidden", nil) }

// ResetConfirmTitle, ResetConfirmBody, and ResetConfirmCancel drive the
// two-step Reset confirmation.
func ResetConfirmTitle(locale string) string  { return T(locale, "host.reset_confirm_title", nil) }
func ResetConfirmBody(locale string) string   { return T(locale, "host.reset_confirm_body", nil) }
func ResetConfirmCancel(locale string) string { return T(locale, "host.reset_confirm_cancel", nil) }

// PhaseLine, SpotlightLine, and TurnTimerLine render the run-status detail
// lines through the catalog instead of raw concatenation.
func PhaseLine(locale, phase string) string {
	return T(locale, "host.status.phase_line", map[string]string{"phase": phase})
}

func SpotlightLine(locale, seat string) string {
	return T(locale, "host.status.spotlight_line", map[string]string{"seat": seat})
}

func TurnTimerLine(locale, seat, remainingMs, totalMs string) string {
	return T(locale, "host.status.turn_timer_line", map[string]string{"seat": seat, "ms": remainingMs, "total": totalMs})
}

// StatusPhaseLabel, StatusTurnLabel, and StatusSeatsLabel caption the live
// status strip's pills.
func StatusPhaseLabel(locale string) string { return T(locale, "host.status.phase", nil) }
func StatusTurnLabel(locale string) string  { return T(locale, "host.status.turn", nil) }
func StatusSeatsLabel(locale string) string { return T(locale, "host.status.seats", nil) }

// StatusSeatsValue renders "{joined}/{total}" through the catalog.
func StatusSeatsValue(locale string, joined, total int) string {
	return T(locale, "host.status.seats_value", map[string]string{"joined": strconv.Itoa(joined), "total": strconv.Itoa(total)})
}

// StatusTimersOn and StatusTimersOff label the timers pill.
func StatusTimersOn(locale string) string  { return T(locale, "host.status.timers_on", nil) }
func StatusTimersOff(locale string) string { return T(locale, "host.status.timers_off", nil) }

// StatusAllClear and StatusFailures label the failures pill.
func StatusAllClear(locale string) string { return T(locale, "host.status.ok", nil) }
func StatusFailures(locale string, n int) string {
	return T(locale, "host.status.failures", map[string]string{"n": strconv.Itoa(n)})
}

// StatusPaused labels the paused pill.
func StatusPaused(locale string) string { return T(locale, "ui.dm.paused", nil) }

// StatusTurnNone is the placeholder shown when no seat has the spotlight.
func StatusTurnNone(locale string) string { return T(locale, "host.status.turn_none", nil) }
