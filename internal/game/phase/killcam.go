package phase

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// SetTime supplies monotonic engine time for cinematic reconnect seeks. Paused
// intervals are excluded; the room's pausable timer owns playback completion.
func (m *Machine) SetTime(at time.Duration) {
	if at > m.now && m.killcam.URL != "" && !m.paused {
		m.killcamTime += at - m.now
	}
	if at > m.now {
		m.now = at
	}
}

func (m *Machine) beginKillcam(seat int, outcome string) []domain.Effect {
	pc, ok := m.combat.Participant(seat)
	if !ok || m.killcam.URL != "" {
		return nil
	}
	asset := m.oneShot.KillCams[strings.ToLower(string(pc.Build.Class))+"_"+outcome]
	if asset.URL == "" || asset.DurationMS != 4000 {
		return nil
	}
	m.killcamSequence++
	attacker, victim := pc.Name, "Drowned Thrall"
	if attacker == "" {
		attacker = pc.ID
	}
	if outcome == "defeat" {
		attacker, victim = victim, attacker
	}
	m.killcam = domain.KillCamView{URL: asset.URL, Sequence: m.killcamSequence, Outcome: outcome, Attacker: attacker, Victim: victim, DurationMS: 4000, Playing: true}
	m.killcamTime = 0
	return []domain.Effect{domain.StartTimer{Name: m.killcamTimer(), After: 4 * time.Second, Pausable: true, Scope: domain.Scope{Machine: vocab.MachineSession}}}
}

func (m Machine) killcamTimer() string { return fmt.Sprintf("killcam:%d", m.killcam.Sequence) }

func (m *Machine) stepKillcam(event domain.Event) (Result, bool, error) {
	if m.killcam.URL == "" {
		return Result{}, false, nil
	}
	if timer, ok := event.(domain.TimerFired); ok && timer.Name == m.killcamTimer() {
		m.killcam = domain.KillCamView{}
		m.killcamTime = 0
		if m.combat.Phase == combat.Done {
			m.syncCombatSeats()
			result, err := m.transition(eventCliffhanger, nil)
			return result, true, err
		}
		return Result{}, true, nil
	}
	if _, ok := event.(domain.Act); ok {
		return Result{}, true, errors.New("combat cinematic is playing")
	}
	// Delayed narration and turn callbacks cannot cut off the cinematic.
	// The fixed combat deadline is handled first by stepCombat.
	return Result{}, true, nil
}

func (m Machine) decorateKillcam(view *domain.View) {
	keys := make([]string, 0, len(m.oneShot.KillCams))
	for key := range m.oneShot.KillCams {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if asset := m.oneShot.KillCams[key]; asset.URL != "" {
			view.Preload = append(view.Preload, asset.URL)
		}
	}
	if m.State() != vocab.StateCombat || m.killcam.URL == "" {
		return
	}
	view.KillCam = m.killcam
	view.KillCam.OffsetMS = min(m.killcamTime.Milliseconds(), view.KillCam.DurationMS)
	view.KillCam.Playing = !m.paused
}
