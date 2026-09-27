package host

import (
	"net/url"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// lobbyPhase and endPhase are the engine phase keys that gate Start and mark
// the run as finished; every other phase counts as "in play" for host button
// prominence (§0.13 stage runbook).
const (
	lobbyPhase = "lobby"
	endPhase   = "end"
)

func isLobbyPhase(phase string) bool { return phase == "" || phase == lobbyPhase }
func isEndPhase(phase string) bool   { return phase == endPhase }
func isPlayPhase(phase string) bool  { return !isLobbyPhase(phase) && !isEndPhase(phase) }

type hostAction struct {
	Label   string
	Command df.HostCommandKind
	D20     int32
	Toggle  bool
}

var hostActions = []hostAction{
	{Label: "Start", Command: df.HostCommandKind_HOST_COMMAND_KIND_START},
	{Label: "Pause", Command: df.HostCommandKind_HOST_COMMAND_KIND_PAUSE},
	{Label: "Resume", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESUME},
	{Label: "Skip", Command: df.HostCommandKind_HOST_COMMAND_KIND_SKIP},
	{Label: "Reset", Command: df.HostCommandKind_HOST_COMMAND_KIND_RESET},
	{Label: "Force d20 = 1", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 1},
	{Label: "Force d20 = 20", Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: 20},
	{Label: "Safe Mode", Command: df.HostCommandKind_HOST_COMMAND_KIND_SAFE_MODE, Toggle: true},
	{Label: "Turn timers", Command: df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF, Toggle: true},
	{Label: "Splat", Command: df.HostCommandKind_HOST_COMMAND_KIND_SPLAT_OFF, Toggle: true},
}

func commandFor(action hostAction, token string) *df.HostCommand {
	return &df.HostCommand{HostToken: token, Command: action.Command, D20: action.D20}
}

func actionLabels() []string {
	labels := make([]string, 0, len(hostActions))
	for _, action := range hostActions {
		labels = append(labels, action.Label)
	}
	return labels
}

type hostSnapshot struct {
	State       *df.ScreenState
	View        *df.HostView
	Status      string
	SafeMode    bool
	TimersOn    bool
	SplatOn     bool
	Connected   bool
	Locale      string
	RoomLocale  string
	Selector    RoomLocaleSelector
	Phase       string
	Spotlight   string
	TurnSeat    string
	TurnMs      int64
	TurnTotal   int64
	Paused      bool
	SeatsJoined int
	SeatsTotal  int
	Failures    int
}

type testerLinks struct {
	DM    string
	Phone string
	Host  string
}

func linksFor(origin, hostToken, dmToken, room string) testerLinks {
	base := strings.TrimRight(origin, "/")
	dmQuery := ""
	if dmToken != "" {
		dmQuery = "?token=" + url.QueryEscape(dmToken)
	}
	phoneQuery := ""
	if room != "" {
		phoneQuery = "?room=" + url.QueryEscape(room)
	}
	hostQuery := ""
	if hostToken != "" {
		hostQuery = "?t=" + url.QueryEscape(hostToken)
	}
	return testerLinks{DM: base + "/dm" + dmQuery, Phone: base + "/p" + phoneQuery, Host: base + "/host" + hostQuery}
}

// linksFromState accepts URL fields added by a later server contract without
// coupling this lane to an uncommitted generated-proto change. Empty fields
// fall back to the stable host-page inputs supplied by the browser.
func linksFromState(origin, hostToken, dmToken, room string, state *df.ScreenState) testerLinks {
	links := linksFor(origin, hostToken, dmToken, room)
	view := state.GetHost()
	if view == nil {
		return links
	}
	fields := view.ProtoReflect().Descriptor().Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		value := view.ProtoReflect().Get(field)
		if field.Kind() != protoreflect.StringKind || value.String() == "" {
			continue
		}
		switch string(field.Name()) {
		case "dm_url":
			links.DM = value.String()
		case "phone_url", "join_url":
			links.Phone = value.String()
		case "host_url":
			links.Host = value.String()
		case "dm_token":
			links.DM = linksFor(origin, hostToken, value.String(), room).DM
		case "room_code":
			links.Phone = linksFor(origin, hostToken, dmToken, value.String()).Phone
		}
	}
	return links
}

func snapshotFromState(state *df.ScreenState) hostSnapshot {
	snapshot := hostSnapshot{State: state, Status: T("en", "ui.host.waiting", nil), Locale: "en", RoomLocale: "en", Selector: NewRoomLocaleSelector("en")}
	if state == nil {
		return snapshot
	}
	snapshot.Phase = state.GetPhase()
	snapshot.Spotlight = state.GetSpotlightSeat()
	snapshot.Paused = state.GetPaused()
	if state.GetHost() != nil {
		snapshot.View = state.GetHost()
		snapshot.TimersOn = state.GetHost().GetTurnTimersEnabled()
		snapshot.Locale = localeOrDefault(state.GetHost().GetLocale())
		snapshot.RoomLocale = localeOrDefault(state.GetHost().GetRoomLocale())
		snapshot.Selector = NewRoomLocaleSelector(snapshot.RoomLocale)
		snapshot.Status = state.GetHost().GetRunMode()
		if snapshot.Status == "" {
			snapshot.Status = state.GetPhase()
		}
	}
	if state.GetPaused() {
		snapshot.Status = T(snapshot.Locale, "ui.dm.paused", nil)
	}
	snapshot.Connected = true
	if host := state.GetHost(); host != nil {
		if dm := host.GetDm(); dm != nil {
			if dm.GetTurnTimer() != nil {
				timer := dm.GetTurnTimer()
				snapshot.TurnSeat = timer.GetSeat()
				snapshot.TurnMs = timer.GetRemainingMs()
				snapshot.TurnTotal = timer.GetTotalMs()
			}
			seats := dm.GetSeats()
			snapshot.SeatsTotal = len(seats)
			for _, seat := range seats {
				if seat.GetJoined() {
					snapshot.SeatsJoined++
				}
			}
		}
		snapshot.Failures = countFailures(host.GetLogTail())
	}
	return snapshot
}

// countFailures scans the host's log tail for lines that look like a
// failure, so the status strip can surface them without a dedicated
// server-side failure counter.
func countFailures(lines []string) int {
	count := 0
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
			count++
		}
	}
	return count
}

// humanizePhase turns an engine phase key (for example "hook_event") into a
// display word ("Hook event") without hard-coding a translation per phase.
func humanizePhase(phase string) string {
	if phase == "" {
		return ""
	}
	phase = strings.ReplaceAll(phase, "_", " ")
	return strings.ToUpper(phase[:1]) + phase[1:]
}

// maskLinkToken hides a token or seat-token query value in a copyable link,
// returning the masked display string and whether anything was masked. The
// caller keeps the real value for copy-to-clipboard and reveal-on-hold.
func maskLinkToken(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw, false
	}
	query := parsed.Query()
	for _, key := range []string{"t", "token"} {
		value := query.Get(key)
		if value == "" {
			continue
		}
		encoded := "=" + url.QueryEscape(value)
		masked := strings.Replace(raw, encoded, "="+strings.Repeat("•", 8), 1)
		if masked != raw {
			return masked, true
		}
	}
	return raw, false
}

func runStatusLines(snapshot hostSnapshot) []string {
	if snapshot.View == nil {
		return []string{NoSnapshot(snapshot.Locale)}
	}
	lines := []string{}
	if mode := snapshot.View.GetRunMode(); mode != "" {
		lines = append(lines, ModeLine(snapshot.Locale, mode))
	}
	if snapshot.Phase != "" {
		lines = append(lines, PhaseLine(snapshot.Locale, humanizePhase(snapshot.Phase)))
	}
	if snapshot.Spotlight != "" {
		lines = append(lines, SpotlightLine(snapshot.Locale, snapshot.Spotlight))
	}
	if snapshot.TurnSeat != "" || snapshot.TurnMs != 0 || snapshot.TurnTotal != 0 {
		lines = append(lines, TurnTimerLine(snapshot.Locale, snapshot.TurnSeat, strconv.FormatInt(snapshot.TurnMs, 10), strconv.FormatInt(snapshot.TurnTotal, 10)))
	}
	lines = append(lines, NextD20Line(snapshot.Locale, snapshot.View.GetNextD20()), CombatCapLine(snapshot.Locale, snapshot.View.GetCombatCapRemainingMs()))
	return lines
}

func commandForToggle(action hostAction, token string, on bool) *df.HostCommand {
	command := commandFor(action, token)
	command.On = on
	if action.Command == df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_OFF && on {
		command.Command = df.HostCommandKind_HOST_COMMAND_KIND_TIMERS_ON
	}
	return command
}
