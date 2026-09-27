package media

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// BillboardLoopsConfig wires the GenerateBillboardLoops executor. Each loop
// resolves build-time catalogue → cache → generation; Generate false stops
// at the cache (the fake and safe modes, or no fal key).
type BillboardLoopsConfig struct {
	Generator  *BillboardGenerator
	Generate   bool
	Catalogue  func(seat domain.SeatID, action string) (domain.Asset, bool)
	Subject    func(context.Context, domain.SeatID) (BillboardSubject, error)
	LevelStill func(context.Context) ([]byte, error)
	OnResult   func(slot string, result BillboardResult, err error)
}

// BillboardLoopsExecutor runs one seat's loops and posts AssetReady or
// AssetFailed per loop slot.
type BillboardLoopsExecutor struct {
	config BillboardLoopsConfig
}

// NewBillboardLoopsExecutor constructs the executor.
func NewBillboardLoopsExecutor(config BillboardLoopsConfig) *BillboardLoopsExecutor {
	return &BillboardLoopsExecutor{config: config}
}

// BillboardSlot names the asset slot of one loop: billboard:<seat>:<action>.
func BillboardSlot(seat domain.SeatID, action string) string {
	return fmt.Sprintf("billboard:%d:%s", seat, action)
}

// Execute resolves the requested loops in order: catalogue, then cache, then
// a budget reservation, all in the effect's action order so a tight cap
// admits idle before attack before hit. The admitted renders then run in
// parallel; the vendor pool bounds how many render at once.
func (e *BillboardLoopsExecutor) Execute(ctx context.Context, effect domain.GenerateBillboardLoops, scope domain.Scope, in ports.Inbox) {
	var group sync.WaitGroup
	inputs := e.inputs(effect.Seat)
	for _, action := range effect.Clips {
		slot := BillboardSlot(effect.Seat, action)
		if e.config.Catalogue != nil {
			if asset, ok := e.config.Catalogue(effect.Seat, action); ok {
				post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: slot, Asset: asset}})
				continue
			}
		}
		spec, reservation, result, err := e.prepare(ctx, inputs, action)
		if err != nil || result.Cached {
			e.finish(ctx, in, scope, slot, result, err)
			continue
		}
		group.Go(func() {
			rendered, renderErr := e.config.Generator.Render(ctx, spec, reservation)
			e.finish(ctx, in, scope, slot, rendered, renderErr)
		})
	}
	group.Wait()
}

// loopInputs loads the seat's subject and the level still once per effect.
type loopInputs struct {
	once    sync.Once
	subject BillboardSubject
	level   []byte
	err     error
	load    func(context.Context) (BillboardSubject, []byte, error)
}

func (e *BillboardLoopsExecutor) inputs(seat domain.SeatID) *loopInputs {
	return &loopInputs{load: func(ctx context.Context) (BillboardSubject, []byte, error) {
		if e.config.Generator == nil || e.config.Subject == nil || e.config.LevelStill == nil {
			return BillboardSubject{}, nil, ErrBillboardDisabled
		}
		subject, err := e.config.Subject(ctx, seat)
		if err != nil {
			return BillboardSubject{}, nil, err
		}
		level, err := e.config.LevelStill(ctx)
		return subject, level, err
	}}
}

func (e *BillboardLoopsExecutor) prepare(ctx context.Context, inputs *loopInputs, action string) (BillboardSpec, *budget.Reservation, BillboardResult, error) {
	if !BillboardActions(action) {
		return BillboardSpec{}, nil, BillboardResult{}, fmt.Errorf("media: unknown billboard action %q", action)
	}
	inputs.once.Do(func() { inputs.subject, inputs.level, inputs.err = inputs.load(ctx) })
	if inputs.err != nil {
		return BillboardSpec{}, nil, BillboardResult{}, inputs.err
	}
	spec := BillboardSpec{Action: action, Subject: inputs.subject, LevelStill: inputs.level}
	if e.config.Generate {
		reservation, result, err := e.config.Generator.Prepare(ctx, spec)
		return spec, reservation, result, err
	}
	asset, ok, err := e.config.Generator.Lookup(ctx, spec)
	if err != nil {
		return spec, nil, BillboardResult{}, err
	}
	if !ok {
		return spec, nil, BillboardResult{}, ErrBillboardDisabled
	}
	return spec, nil, BillboardResult{Asset: asset, Key: e.config.Generator.Key(spec), Cached: true}, nil
}

func (e *BillboardLoopsExecutor) finish(ctx context.Context, in ports.Inbox, scope domain.Scope, slot string, result BillboardResult, err error) {
	if e.config.OnResult != nil {
		e.config.OnResult(slot, result, err)
	}
	if err != nil {
		post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: slot, FailureKind: billboardFailure(err)}})
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: slot, Asset: result.Asset}})
}

func billboardFailure(err error) vocab.ErrKind {
	switch {
	case errors.Is(err, budget.ErrCapReached):
		return vocab.ErrRateLimited
	case errors.Is(err, ErrBillboardDeadline):
		return vocab.ErrTimeout
	case errors.Is(err, ErrBillboardDisabled):
		return vocab.ErrUnavailable
	default:
		return failureKind(err)
	}
}
