package nested

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// SlotState is the lifecycle state of an asset slot.
type SlotState string

const (
	// Pending means the primary asset is still being generated.
	SlotPending SlotState = "pending"
	// Ready means the slot contains the requested asset.
	SlotReady SlotState = "ready"
	// Failed means no asset is available, including after exhausting fallbacks.
	SlotFailed SlotState = "failed"
	// Fallback means the slot is serving an ordered fallback asset.
	SlotFallback SlotState = "fallback"
)

const eventFallback vocab.EventKind = "slot_fallback"

// SlotMachine tracks one asset and its ordered fallback chain. It is pure and is
// intended to be owned by the engine loop.
type SlotMachine struct {
	name          string
	state         SlotState
	asset         domain.Asset
	fallbacks     []domain.Asset
	fallbackIndex int
	transitions   fsm.Machine
}

// NewSlot creates a pending slot. Fallbacks are tried from left to right after a
// failure or deadline; the supplied slice is copied.
func NewSlot(name string, fallbacks []domain.Asset) (SlotMachine, error) {
	if name == "" {
		return SlotMachine{}, errors.New("slot name is empty")
	}
	definition := fsm.Def{
		Initial: vocab.StateID(SlotPending),
		States:  []vocab.StateID{vocab.StateID(SlotPending), vocab.StateID(SlotReady), vocab.StateID(SlotFailed), vocab.StateID(SlotFallback)},
		Transitions: []fsm.Transition{
			{From: vocab.StateID(SlotPending), Event: vocab.EventAssetPartial, To: vocab.StateID(SlotPending)},
			{From: vocab.StateID(SlotPending), Event: vocab.EventAssetReady, To: vocab.StateID(SlotReady)},
			{From: vocab.StateID(SlotPending), Event: vocab.EventAssetFailed, To: vocab.StateID(SlotFailed)},
			{From: vocab.StateID(SlotPending), Event: vocab.EventTimerFired, To: vocab.StateID(SlotFailed)},
			{From: vocab.StateID(SlotFailed), Event: eventFallback, To: vocab.StateID(SlotFallback)},
			{From: vocab.StateID(SlotFallback), Event: vocab.EventAssetFailed, To: vocab.StateID(SlotFailed)},
		},
	}
	transitions, err := fsm.New(definition)
	if err != nil {
		return SlotMachine{}, err
	}
	return SlotMachine{name: name, state: SlotPending, fallbacks: append([]domain.Asset(nil), fallbacks...), transitions: transitions}, nil
}

// Name returns the logical slot name.
func (m SlotMachine) Name() string { return m.name }

// State returns the current slot lifecycle state.
func (m SlotMachine) State() SlotState { return m.state }

// Asset returns the asset currently served by the slot, if any.
func (m SlotMachine) Asset() (domain.Asset, bool) {
	if m.state != SlotReady && m.state != SlotFallback {
		return domain.Asset{}, false
	}
	return m.asset, m.asset.ID != ""
}

// View returns the host-facing projection of the slot.
func (m SlotMachine) View() domain.SlotView {
	return domain.SlotView{Name: m.name, State: string(m.state), Asset: m.asset.ID}
}

// Step applies one asset lifecycle event. Results arriving after Ready or
// after a terminal Failed state are ignored, which makes late callbacks safe.
func (m *SlotMachine) Step(event domain.Event) error {
	if m == nil {
		return errors.New("slot machine is nil")
	}
	if event == nil {
		return errors.New("slot event is nil")
	}
	kind, slot, asset, ok := slotEvent(event)
	if !ok {
		return errors.New("event is not an asset slot event")
	}
	if slot != "" && slot != m.name {
		return nil
	}
	if m.state == SlotReady || (m.state == SlotFailed && kind != vocab.EventAssetFailed) {
		return nil
	}
	if m.state == SlotFallback && kind == vocab.EventAssetReady {
		return nil
	}
	if kind == vocab.EventTimerFired && m.state != SlotPending {
		return nil
	}
	if kind == vocab.EventAssetPartial {
		_, err := m.transitions.Step(kind)
		return err
	}
	result, err := m.transitions.Step(kind)
	if err != nil {
		return nil
	}
	if kind == vocab.EventAssetReady {
		m.asset = asset
		m.state = SlotReady
		return nil
	}
	if result.To == vocab.StateID(SlotFailed) {
		m.state = SlotFailed
		m.asset = domain.Asset{}
		if m.nextFallback() {
			_, err = m.transitions.Step(eventFallback)
			if err != nil {
				return err
			}
			m.state = SlotFallback
		}
		return nil
	}
	return nil
}

func (m *SlotMachine) nextFallback() bool {
	if m.fallbackIndex >= len(m.fallbacks) {
		return false
	}
	m.asset = m.fallbacks[m.fallbackIndex]
	m.fallbackIndex++
	return m.asset.ID != ""
}

func slotEvent(event domain.Event) (vocab.EventKind, string, domain.Asset, bool) {
	switch value := event.(type) {
	case domain.AssetPartial:
		return vocab.EventAssetPartial, value.Slot, value.Asset, true
	case domain.AssetReady:
		return vocab.EventAssetReady, value.Slot, value.Asset, true
	case domain.AssetFailed:
		return vocab.EventAssetFailed, value.Slot, domain.Asset{}, true
	case domain.TimerFired:
		return vocab.EventTimerFired, "", domain.Asset{}, true
	default:
		return "", "", domain.Asset{}, false
	}
}
