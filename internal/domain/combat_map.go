package domain

// CombatMapView is the engine's top-down movement map for one seat in
// Combat: the content grid, every token, and, for the active seat only, the
// cells it may walk to with the exact engine path to each. The phone draws
// it and never pathfinds (§0.21.3 "Phone moves", R-D3, R-D8).
type CombatMapView struct {
	Cols, Rows int
	Walkable   []Cell
	Tokens     []CombatMapToken
	// Me is the seat's own token; empty for a seat without one.
	Me TokenID
	// MoveLeft is the normal movement left this turn, in cells.
	MoveLeft int
	// CanDash is true while the seat's action is unused, so a Dash may go
	// up to MoveLeft + 6 cells and end the turn.
	CanDash bool
	// Reach lists the destinations; empty for the watching seat.
	Reach []CombatMapReach
	// WalkStepMS and DashStepMS are the renderer paces per path cell.
	WalkStepMS, DashStepMS int
}

// CombatMapToken is one combatant on the map.
type CombatMapToken struct {
	ID        TokenID
	Seat      SeatID
	Kind      string
	Cell      Cell
	HP, HPMax int
	Enemy     bool
	Down      bool
	Active    bool
	// Path, Anim, AnimSeq, and StepMS describe the latest walk so clients
	// animate it once per AnimSeq, like the TV battle stage.
	Path    []Cell
	Anim    string
	AnimSeq uint64
	StepMS  int
}

// CombatMapReach is one legal destination and the path the engine walks.
// Dash marks a cell only a Dash reaches.
type CombatMapReach struct {
	Cell Cell
	Path []Cell
	Dash bool
}
