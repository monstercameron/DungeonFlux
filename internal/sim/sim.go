package sim

import (
	"fmt"
	"sort"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Engine is the pure engine surface needed by Simulator.
type Engine interface {
	Step(domain.Envelope) domain.StepOut
	Inspect() domain.Inspect
}

// Outcome describes the result of one asynchronous effect. Event is delivered
// after After. A silent outcome deliberately delivers nothing.
type Outcome struct {
	Kind    vocab.EffectKind
	After   time.Duration
	Event   domain.Event
	Silence bool
}

// Success creates an outcome that delivers event after delay.
func Success(kind vocab.EffectKind, delay time.Duration, event domain.Event) Outcome {
	return Outcome{Kind: kind, After: delay, Event: event}
}

// Error creates an outcome for a failure event, such as LineFailed or
// InterpretFailed. The engine decides the fallback behaviour.
func Error(kind vocab.EffectKind, delay time.Duration, event domain.Event) Outcome {
	return Outcome{Kind: kind, After: delay, Event: event}
}

// Silence creates an outcome that never produces an event.
func Silence(kind vocab.EffectKind) Outcome {
	return Outcome{Kind: kind, Silence: true}
}

// Script supplies asynchronous outcomes in effect order. An outcome matches
// the next effect with the same Kind; unmatched effects are left silent.
type Script struct {
	Outcomes []Outcome
}

// Result is the observable result of a simulator run.
type Result struct {
	Now        time.Duration
	Steps      int
	Envelopes  []domain.Envelope
	LastOutput domain.StepOut
}

// Simulator drives an engine with a virtual clock and bounded, deterministic
// event ordering. It is intended for walk tests and replay checks.
type Simulator struct {
	engine    Engine
	timers    fsm.Timers
	now       time.Duration
	seq       uint64
	queue     []scheduled
	outcomes  []Outcome
	envelopes []domain.Envelope
	steps     int
	timerIDs  map[string]fsm.Scope
	timerDue  map[string]time.Duration
	last      domain.StepOut
}

type scheduled struct {
	at    time.Duration
	order uint64
	event domain.Event
}

// New creates a simulator with a scripted effect sequence.
func New(engine Engine, script Script) *Simulator {
	return &Simulator{engine: engine, timers: fsm.NewTimers(), outcomes: append([]Outcome(nil), script.Outcomes...), timerIDs: make(map[string]fsm.Scope), timerDue: make(map[string]time.Duration)}
}

// Now reports the current virtual time.
func (s *Simulator) Now() time.Duration { return s.now }

// Send applies an external event at the current virtual time.
func (s *Simulator) Send(event domain.Event) domain.StepOut {
	return s.apply(event)
}

// Advance moves virtual time and applies every event due by the new time.
func (s *Simulator) Advance(delta time.Duration) (Result, error) {
	if delta < 0 {
		return s.result(), fmt.Errorf("simulator: negative advance %s", delta)
	}
	s.now += delta
	fired, _ := s.timers.Advance(delta)
	for _, timer := range fired {
		delete(s.timerIDs, timer.Name)
		delete(s.timerDue, timer.Name)
		s.enqueue(0, domain.TimerFired{Name: timer.Name})
	}
	for len(s.queue) > 0 && s.queue[0].at <= s.now {
		next := s.queue[0]
		s.queue = s.queue[1:]
		s.apply(next.event)
	}
	return s.result(), nil
}

// RunUntilEnd feeds the initial event, then advances to each queued event
// until the engine reaches End, the queue is empty, or max is exceeded.
func (s *Simulator) RunUntilEnd(initial domain.Event, max time.Duration) (Result, error) {
	if max < 0 {
		return s.result(), fmt.Errorf("simulator: negative limit %s", max)
	}
	s.Send(initial)
	for s.engine.Inspect().Path != string(vocab.StateEnd) && (len(s.queue) > 0 || len(s.timerIDs) > 0) {
		delta := s.nextDelta()
		if s.now+delta > max {
			return s.result(), fmt.Errorf("simulator: virtual time limit %s exceeded", max)
		}
		if _, err := s.Advance(delta); err != nil {
			return s.result(), err
		}
		if delta == 0 && s.engine.Inspect().Path != string(vocab.StateEnd) && len(s.queue) == 0 {
			return s.result(), fmt.Errorf("simulator: no runnable event")
		}
	}
	if s.engine.Inspect().Path != string(vocab.StateEnd) {
		return s.result(), fmt.Errorf("simulator: ended before End (queue empty)")
	}
	return s.result(), nil
}

func (s *Simulator) apply(event domain.Event) domain.StepOut {
	s.seq++
	env := domain.Envelope{Seq: s.seq, At: s.now, Event: event}
	out := s.engine.Step(env)
	s.steps++
	s.last = out
	s.envelopes = append(s.envelopes, env)
	s.consume(out.Effects)
	return out
}

func (s *Simulator) consume(effects []domain.Effect) {
	for _, effect := range effects {
		switch value := effect.(type) {
		case domain.StartTimer:
			s.startTimer(domain.TimerEffect(value))
		case domain.TimerEffect:
			s.startTimer(value)
		case domain.CancelTimer:
			s.cancelTimer(value.Name)
		case domain.FreezeTimer:
			s.freezeTimer(value.Name)
		case domain.ThawTimer:
			s.thawTimer(value.Name)
		default:
			s.scheduleOutcome(effect.Kind())
		}
	}
}

func (s *Simulator) cancelTimer(name string) {
	if scope, ok := s.timerIDs[name]; ok {
		_, _ = s.timers.CancelTimer(scope, name)
		delete(s.timerIDs, name)
		delete(s.timerDue, name)
	}
}

func (s *Simulator) freezeTimer(name string) {
	if scope, ok := s.timerIDs[name]; ok {
		_, _ = s.timers.FreezeTimer(scope, name)
	}
}

func (s *Simulator) thawTimer(name string) {
	if scope, ok := s.timerIDs[name]; ok {
		_, _ = s.timers.ThawTimer(scope, name)
	}
}

func (s *Simulator) startTimer(effect domain.TimerEffect) {
	scope := fsm.Scope{Machine: effect.Scope.Machine, Key: effect.Scope.Key, Epoch: effect.Scope.Epoch}
	if _, err := s.timers.StartTimer(scope, effect.Name, effect.After); err != nil {
		return
	}
	s.timerIDs[effect.Name] = scope
	s.timerDue[effect.Name] = s.now + effect.After
}

func (s *Simulator) nextDelta() time.Duration {
	if len(s.queue) > 0 {
		return s.queue[0].at - s.now
	}
	var due time.Duration
	for _, value := range s.timerDue {
		if due == 0 || value < due {
			due = value
		}
	}
	if due <= s.now {
		return 0
	}
	return due - s.now
}

func (s *Simulator) scheduleOutcome(kind vocab.EffectKind) {
	for index, outcome := range s.outcomes {
		if outcome.Kind != kind {
			continue
		}
		s.outcomes = append(s.outcomes[:index], s.outcomes[index+1:]...)
		if outcome.Silence || outcome.Event == nil {
			return
		}
		s.enqueue(outcome.After, outcome.Event)
		return
	}
}

func (s *Simulator) enqueue(after time.Duration, event domain.Event) {
	s.queue = append(s.queue, scheduled{at: s.now + after, order: s.seq, event: event})
	sort.SliceStable(s.queue, func(i, j int) bool {
		if s.queue[i].at != s.queue[j].at {
			return s.queue[i].at < s.queue[j].at
		}
		return s.queue[i].order < s.queue[j].order
	})
}

func (s *Simulator) result() Result {
	return Result{Now: s.now, Steps: s.steps, Envelopes: append([]domain.Envelope(nil), s.envelopes...), LastOutput: s.last}
}
