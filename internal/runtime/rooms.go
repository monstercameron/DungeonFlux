package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"sort"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// RoomState stores room-owned data that survives creation of a new run.
// Runtime engine state is deliberately not stored here.
type RoomState struct {
	mu         sync.Mutex
	seed       []byte
	seeded     bool
	runIndex   uint64
	seats      map[domain.SeatID]domain.Seat
	splat      *domain.Report
	roomLocale string
	locales    map[domain.SeatID]string
}

// ResetPlan contains the new run seed and events to replay into its engine.
type ResetPlan struct {
	Seed     []byte
	RunIndex uint64
	Joins    []domain.Join
	Splat    *domain.Report
}

// NewRoomState creates room state. A non-empty seed is reused on every reset;
// an empty seed requests a fresh cryptographically random seed per reset.
func NewRoomState(seed []byte) (*RoomState, error) {
	state := &RoomState{seats: make(map[domain.SeatID]domain.Seat)}
	if len(seed) != 0 {
		state.seed = append([]byte(nil), seed...)
		state.seeded = true
		return state, nil
	}
	return state, nil
}

// Join records or refreshes a room-level seat. It is retained across resets.
func (r *RoomState) Join(seat domain.Seat) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.seats == nil {
		r.seats = make(map[domain.SeatID]domain.Seat)
	}
	seat.Character = cloneCharacter(seat.Character)
	r.seats[seat.ID] = seat
	if r.locales == nil {
		r.locales = make(map[domain.SeatID]string)
	}
	if seat.Locale != "" {
		r.locales[seat.ID] = seat.Locale
	}
}

// SetSeatLocale settles a seat locale against the room default and records it.
func (r *RoomState) SetSeatLocale(seat domain.SeatID, requested string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.locales == nil {
		r.locales = make(map[domain.SeatID]string)
	}
	tag := settleLocale(requested, r.roomLocale)
	r.locales[seat] = tag
	if entry, ok := r.seats[seat]; ok {
		entry.Locale = tag
		r.seats[seat] = entry
	}
	return tag
}

// SeatLocale returns the settled locale for a seat or the room default.
func (r *RoomState) SeatLocale(seat domain.SeatID) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tag, ok := r.locales[seat]; ok && tag != "" {
		return tag
	}
	return roomLocaleOrDefault(r.roomLocale)
}

// SetRoomLocale updates the room default locale used for new seats.
func (r *RoomState) SetRoomLocale(locale string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roomLocale = settleLocale(locale, r.roomLocale)
	return roomLocaleOrDefault(r.roomLocale)
}

// RoomLocale returns the room default locale.
func (r *RoomState) RoomLocale() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return roomLocaleOrDefault(r.roomLocale)
}

func settleLocale(requested, fallback string) string {
	norm := func(tag string) string {
		out := ""
		for _, part := range tag {
			if part == '-' || part == '_' || part == ' ' {
				break
			}
			if part >= 'A' && part <= 'Z' {
				part += 'a' - 'A'
			}
			out += string(part)
		}
		return out
	}
	if norm(requested) == "en" || norm(requested) == "es" {
		return norm(requested)
	}
	if norm(fallback) == "en" || norm(fallback) == "es" {
		return norm(fallback)
	}
	if norm(fallback) != "" {
		return norm(fallback)
	}
	return "en"
}

func roomLocaleOrDefault(tag string) string {
	if tag == "" {
		return "en"
	}
	return tag
}

// SetSplat records the latest splat readiness report for the next reset.
func (r *RoomState) SetSplat(report domain.Report) {
	if report.ReportKind != vocab.ReportSplatReady && report.ReportKind != vocab.ReportSplatFailed {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.splat = &domain.Report{ReportKind: report.ReportKind, ID: report.ID}
}

// Reset advances the run and returns the join/report events to post into the
// replacement engine. The same explicit seed produces the same seed forever.
func (r *RoomState) Reset() (ResetPlan, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runIndex++
	seed, err := r.nextSeedLocked()
	if err != nil {
		return ResetPlan{}, err
	}
	plan := ResetPlan{Seed: seed, RunIndex: r.runIndex}
	ids := make([]int, 0, len(r.seats))
	for id := range r.seats {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	for _, id := range ids {
		plan.Joins = append(plan.Joins, domain.Join{Seat: domain.SeatID(id), JoinKind: "phone", Locale: r.locales[domain.SeatID(id)]})
	}
	if r.splat != nil {
		plan.Splat = &domain.Report{ReportKind: r.splat.ReportKind, ID: r.splat.ID}
	}
	return plan, nil
}

// Seed returns a copy of the current deterministic rehearsal seed, if any.
func (r *RoomState) Seed() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.seed...)
}

func (r *RoomState) nextSeedLocked() ([]byte, error) {
	if r.seeded {
		return deriveSeed(r.seed), nil
	}
	seed := make([]byte, sha256.Size)
	if _, err := rand.Read(seed); err != nil {
		return nil, errors.Join(errors.New("generate run seed"), err)
	}
	return seed, nil
}

func deriveSeed(seed []byte) []byte {
	h := sha256.New()
	_, _ = h.Write(seed)
	var zero [4]byte
	_, _ = h.Write(zero[:])
	return h.Sum(nil)
}

func cloneCharacter(character *domain.Character) *domain.Character {
	if character == nil {
		return nil
	}
	clone := *character
	return &clone
}
