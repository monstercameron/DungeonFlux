package combat

import (
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

// Phase identifies the externally visible state of a combat child machine.
type Phase string

const (
	// Intro is the short setup state before the first player acts.
	Intro Phase = "intro"
	// PCTurn is the state in which one player may submit a move.
	PCTurn Phase = "pc_turn"
	// Rolling is the attack resolution state.
	Rolling Phase = "rolling"
	// EnemyTurn is the drowned thrall's turn.
	EnemyTurn Phase = "enemy_turn"
	// Done is the terminal combat state.
	Done Phase = "done"
)

// Cell identifies a battlefield grid cell. Coordinates are zero based.
type Cell struct {
	X int
	Y int
}

// Grid describes the cells available to combat movement.
type Grid struct {
	Cols     int
	Rows     int
	Walkable map[Cell]bool
}

// TokenPresentation contains renderer-only facts for the latest token action.
type TokenPresentation struct {
	Path    []Cell
	Anim    string
	AnimSeq uint64
	// StepMS is the walk pace in milliseconds per path cell; zero means the
	// default WalkStepMS. A Dash walks faster (DashStepMS, R-D8).
	StepMS int64
}

// CameraPresentation describes the engine-selected combat camera.
type CameraPresentation struct {
	Preset       string
	FocusTokenID string
	Follow       bool
	DurationMS   int64
	Seq          uint64
}

// Highlight contains an engine-selected set of cells for the renderer.
type Highlight struct {
	Kind  string
	Cells []Cell
}

// ShakePresentation is a one-shot camera impact request.
type ShakePresentation struct {
	AmplitudePX float64
	DurationMS  int64
	Seq         uint64
}

// Presentation is the deterministic combat rendering state.
type Presentation struct {
	Tokens         map[string]TokenPresentation
	Highlights     []Highlight
	Camera         CameraPresentation
	Shake          ShakePresentation
	ContactMS      int64
	ContactTotalMS int64
}

// IsWalkable reports whether cell is inside the grid and available for travel.
func (g Grid) IsWalkable(cell Cell) bool {
	if cell.X < 0 || cell.Y < 0 || cell.X >= g.Cols || cell.Y >= g.Rows {
		return false
	}
	return g.Walkable == nil || g.Walkable[cell]
}

// Participant is the mutable combat projection of one player character.
type Participant struct {
	Seat       int
	ID         string
	Build      rules.Build
	Position   Cell
	HP         int
	MaxHP      int
	AC         int
	Conditions []rules.Condition
	ActionUsed bool
	// Moved counts the cells walked this turn (R-D3, R-D8).
	Moved int
	// Dashed records that the Dash action spent this turn's action (R-D8).
	Dashed bool
}

// IsDown reports whether the participant is unable to take a turn.
func (p Participant) IsDown() bool {
	for _, condition := range p.Conditions {
		if condition == rules.Down || condition == rules.Defeated {
			return true
		}
	}
	return p.HP <= 0
}

// Config contains the initial values for a combat instance.
type Config struct {
	PCs       [2]Participant
	Thrall    rules.CreatureState
	Grid      Grid
	SpawnCell Cell
}

// State is the pure mutable data for one combat child machine.
type State struct {
	Phase          Phase
	PCs            [2]Participant
	Thrall         rules.CreatureState
	Grid           Grid
	ThrallPosition Cell
	TurnSeat       int
	ThrallTurns    int
	TurnNumber     int
	Presentation   Presentation
}

// NewState creates a combat in intro with the fixed PC 1, thrall, PC 2 order.
func NewState(config Config) (State, error) {
	if err := validateConfig(config); err != nil {
		return State{}, err
	}
	state := State{
		Phase: Intro, PCs: config.PCs, Thrall: config.Thrall, Grid: config.Grid,
		ThrallPosition: config.SpawnCell,
	}
	state.initPresentation()
	return state, nil
}

// New is an alias for NewState for callers that construct the child machine.
func New(config Config) (State, error) { return NewState(config) }

// Start enters the first player turn. It is valid only from intro.
func (s *State) Start() error {
	if s == nil {
		return errors.New("combat state is nil")
	}
	if s.Phase != Intro {
		return fmt.Errorf("combat cannot start from %s", s.Phase)
	}
	s.Phase = PCTurn
	s.TurnSeat = 1
	s.TurnNumber = 1
	s.resetAction()
	s.setCamera("TURN_FOCUS", s.PCs[0].ID, true, 500)
	s.setReachHighlights(s.PCs[0])
	return nil
}

// EndPlayerTurn advances PC 1 to the thrall and PC 2 to PC 1.
func (s *State) EndPlayerTurn() error {
	if s == nil {
		return errors.New("combat state is nil")
	}
	if s.Phase != PCTurn || (s.TurnSeat != 1 && s.TurnSeat != 2) {
		return fmt.Errorf("player turn is not active")
	}
	if s.TurnSeat == 1 {
		s.Phase = EnemyTurn
		s.TurnNumber++
		s.clearHighlights()
		s.setCamera("TURN_FOCUS", s.Thrall.ID, true, 500)
		return nil
	}
	s.Phase = PCTurn
	s.TurnSeat = 1
	s.TurnNumber++
	s.resetAction()
	s.setCamera("TURN_FOCUS", s.PCs[0].ID, true, 500)
	s.setReachHighlights(s.PCs[0])
	return nil
}

// EndEnemyTurn advances the thrall to PC 2, then to PC 1.
func (s *State) EndEnemyTurn() error {
	if s == nil {
		return errors.New("combat state is nil")
	}
	if s.Phase != EnemyTurn {
		return fmt.Errorf("enemy turn is not active")
	}
	s.ThrallTurns++
	s.Phase = PCTurn
	s.TurnSeat = 2
	s.TurnNumber++
	s.resetAction()
	s.setCamera("TURN_FOCUS", s.PCs[1].ID, true, 500)
	s.setReachHighlights(s.PCs[1])
	return nil
}

// ActiveParticipant returns a copy of the player whose turn is active.
func (s State) ActiveParticipant() (Participant, bool) {
	if s.Phase != PCTurn || s.TurnSeat < 1 || s.TurnSeat > len(s.PCs) {
		return Participant{}, false
	}
	return s.PCs[s.TurnSeat-1], true
}

// Participant returns a copy of the requested seat.
func (s State) Participant(seat int) (Participant, bool) {
	if seat < 1 || seat > len(s.PCs) {
		return Participant{}, false
	}
	return s.PCs[seat-1], true
}

// SetParticipant replaces a participant while preserving the state value.
func (s *State) SetParticipant(participant Participant) error {
	if s == nil || participant.Seat < 1 || participant.Seat > len(s.PCs) {
		return errors.New("participant seat must be 1 or 2")
	}
	s.PCs[participant.Seat-1] = participant
	return nil
}

func validateConfig(config Config) error {
	for seat, participant := range config.PCs {
		want := seat + 1
		if participant.Seat != want {
			return fmt.Errorf("participant %d has seat %d", want, participant.Seat)
		}
		if participant.ID == "" {
			return fmt.Errorf("participant %d has empty id", want)
		}
		if participant.MaxHP < 1 || participant.HP < 0 || participant.HP > participant.MaxHP {
			return fmt.Errorf("participant %d has invalid hp", want)
		}
	}
	if config.Thrall.ID == "" || config.Thrall.MaxHP < 1 || config.Thrall.HP < 0 || config.Thrall.HP > config.Thrall.MaxHP {
		return errors.New("invalid thrall")
	}
	return nil
}

func (s *State) resetAction() {
	for i := range s.PCs {
		s.PCs[i].ActionUsed = s.PCs[i].IsDown()
		s.PCs[i].Moved, s.PCs[i].Dashed = 0, false
	}
}
