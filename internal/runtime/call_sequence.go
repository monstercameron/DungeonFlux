package runtime

import (
	"context"
	"maps"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type callGroup struct {
	role vocab.Role
	seat domain.SeatID
}

type callReservation struct {
	meta  ports.CallMeta
	epoch uint64
}

type callSequence struct {
	phase vocab.StateID
	epoch uint64
	used  map[callGroup]map[int]uint64
}

type restoredCallKey struct{}

func (r *Room) reserveCall(ctx context.Context, effect domain.Effect, id uint64) (callReservation, bool) {
	if saved, ok := ctx.Value(restoredCallKey{}).(callReservation); ok {
		r.calls.claim(saved, id)
		return saved, true
	}
	view := r.eng.View()
	meta, ok := effectCallMeta(effect, view)
	if !ok {
		return callReservation{}, false
	}
	r.releaseCanceledCalls()
	if r.calls.phase != view.Path || r.calls.used == nil {
		r.calls = callSequence{phase: view.Path, epoch: r.calls.epoch + 1, used: make(map[callGroup]map[int]uint64)}
	}
	meta.Run, meta.Phase = r.run, view.Path
	group := callGroup{role: meta.Role, seat: meta.Seat}
	for r.calls.used[group][meta.Index] != 0 {
		meta.Index++
	}
	call := callReservation{meta: meta, epoch: r.calls.epoch}
	r.calls.claim(call, id)
	return call, true
}

func (s *callSequence) claim(call callReservation, id uint64) {
	if call.epoch != s.epoch || call.meta.Phase != s.phase {
		return
	}
	group := callGroup{role: call.meta.Role, seat: call.meta.Seat}
	if s.used[group] == nil {
		s.used[group] = make(map[int]uint64)
	}
	s.used[group][call.meta.Index] = id
}

func (r *Room) releaseCanceledCalls() {
	for id, work := range r.pending {
		if !work.hasCall || work.ctx.Err() == nil || work.call.epoch != r.calls.epoch {
			continue
		}
		// A scope can end after the worker finished but before its completion
		// marker is drained. That completed call already consumed its position.
		if work.completed != nil && work.completed.Load() {
			continue
		}
		group := callGroup{role: work.call.meta.Role, seat: work.call.meta.Seat}
		if r.calls.used[group][work.call.meta.Index] == id {
			delete(r.calls.used[group], work.call.meta.Index)
		}
	}
}

func (s callSequence) clone() callSequence {
	copy := callSequence{phase: s.phase, epoch: s.epoch, used: make(map[callGroup]map[int]uint64)}
	for group, positions := range s.used {
		copy.used[group] = maps.Clone(positions)
	}
	return copy
}

func effectCallMeta(effect domain.Effect, view domain.View) (ports.CallMeta, bool) {
	meta := ports.CallMeta{}
	switch value := effect.(type) {
	case domain.CharacterFlavor:
		meta.Role, meta.Seat = vocab.RoleCharacterFlavor, value.Seat
	case domain.Interpret:
		meta.Role, meta.Seat, meta.UtteranceID = vocab.RoleInterpret, value.Seat, value.UtteranceID
	case domain.StartLine:
		meta.Role, meta.Seat, meta.UtteranceID = value.Role, view.Spotlight, value.UtteranceID
		meta.Speculative = value.Hold
	case domain.PrerenderText:
		meta.Role, meta.Speculative = value.Role, true
	default:
		return meta, false
	}
	return meta, true
}
