package phone

import (
	"context"
	"sort"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// combatMapState is the phone-local state of the top-down combat map: whether
// the active seat opened it, the cell it picked, and the walk animations it
// is playing. It lives on the CombatModel because the combat screen remounts
// on every snapshot. The engine owns every path; this only picks one.
type combatMapState struct {
	open     bool
	selected *df.Cell
	pending  bool
	walks    map[string]mapWalk
	seen     map[string]mapSeen
	clock    func() float64
}

type mapSeen struct {
	seq  uint64
	cell *df.Cell
}

// mapWalk is one token walk being played: the route (start cell included),
// its pace, and when the phone first saw it.
type mapWalk struct {
	seq     uint64
	route   []*df.Cell
	stepMS  int
	startMS float64
	dash    bool
}

// CombatMapLayout is the render-ready top-down map. Grids wider than tall are
// turned a quarter clockwise so the long side runs down the phone.
type CombatMapLayout struct {
	Cols, Rows int
	Rotated    bool
	// minC and minR are the grid origin of the displayed window: border
	// rows and columns with no walkable cell and no token are trimmed.
	minC, minR  int32
	Cells       []CombatMapCell
	Tokens      []CombatMapToken
	Interactive bool
	Selected    *df.Cell
	Preview     string
	Dash        bool
	MoveLeft    int32
	CanDash     bool
	Pending     bool
}

// CombatMapCell is one displayed grid cell.
type CombatMapCell struct {
	Cell     *df.Cell
	X, Y     int
	Walkable bool
	Reach    bool
	Dash     bool
	Occupied bool
	Selected bool
	PathStep int
}

// CombatMapToken is one displayed token and its running walk, if any.
type CombatMapToken struct {
	ID, Name, Kind, Portrait string
	X, Y                     int
	Me, Enemy, Down, Active  bool
	Walk                     *CombatMapWalk
}

// CombatMapWalk is a running walk: a keyframes rule moving the token from its
// start cell to its cell (translate in cell units), the duration, and how far
// into it the phone already is, so a remounted screen resumes mid-walk.
type CombatMapWalk struct {
	Name       string
	Keyframes  string
	DurationMS int
	ElapsedMS  int
	Dash       bool
}

// OpenMap shows the movement map for the active seat.
func (m *CombatModel) OpenMap() {
	if m != nil {
		m.mapState.open, m.mapState.selected = true, nil
	}
}

// CloseMap returns from the map to the action list.
func (m *CombatModel) CloseMap() {
	if m != nil {
		m.mapState.open, m.mapState.selected = false, nil
	}
}

// MapOpen reports whether the active seat is looking at the movement map.
func (m *CombatModel) MapOpen() bool { return m != nil && m.mapState.open }

// SelectCell picks a destination; a cell the engine did not offer clears
// the selection.
func (m *CombatModel) SelectCell(cell *df.Cell) {
	if m == nil {
		return
	}
	if _, ok := reachPath(m.state.MiniGrid, cell); !ok || m.mapState.pending {
		m.mapState.selected = nil
		return
	}
	m.mapState.selected = cloneMessage(cell)
}

// ClearSelection drops the picked cell and keeps the map open.
func (m *CombatModel) ClearSelection() {
	if m != nil {
		m.mapState.selected = nil
	}
}

// CommitMove sends move{cell} or, for a dash-only cell, dash{cell}.
func (m *CombatModel) CommitMove(ctx context.Context) <-chan ActResult {
	if m == nil || m.mapState.selected == nil {
		return failedAct("pick a cell first")
	}
	entry, ok := reachPath(m.state.MiniGrid, m.mapState.selected)
	if !ok {
		return failedAct("that cell is out of reach")
	}
	moveID := "move"
	if entry.GetDash() {
		moveID = "dash"
	}
	m.mapState.pending = true
	return m.send(ctx, &df.ActRequest{MoveId: moveID, Cell: cloneMessage(entry.GetCell())})
}

// ApplyCommit records the server's answer to a commit.
func (m *CombatModel) ApplyCommit(result ActResult) CombatSnapshot {
	if m == nil {
		return CombatSnapshot{Error: "combat model is unavailable"}
	}
	m.mapState.pending = false
	snapshot := m.ApplyAct(result)
	if snapshot.Error == "" {
		m.mapState.selected = nil
	}
	return snapshot
}

// applyMap folds a new MiniGrid into the map state: it starts a walk for
// each token whose anim_seq advanced with a path, and closes the map when the
// seat can no longer act.
func (s *combatMapState) applyMap(grid *df.MiniGrid, canAct bool, nowMS float64) {
	if !canAct {
		s.open, s.selected, s.pending = false, nil, false
	}
	if s.selected != nil {
		if _, ok := reachPath(grid, s.selected); !ok {
			s.selected = nil
		}
	}
	if s.walks == nil {
		s.walks, s.seen = map[string]mapWalk{}, map[string]mapSeen{}
	}
	for _, token := range grid.GetTokens() {
		id := token.GetTokenId()
		previous, known := s.seen[id]
		s.seen[id] = mapSeen{seq: token.GetAnimSeq(), cell: cloneMessage(token.GetCell())}
		if !known || token.GetAnimSeq() <= previous.seq || len(token.GetPath()) == 0 {
			continue
		}
		route := walkRoute(previous.cell, token.GetPath(), token.GetCell())
		if len(route) < 2 {
			continue
		}
		step := int(token.GetStepMs())
		if step <= 0 {
			step = int(grid.GetWalkStepMs())
		}
		if step <= 0 {
			step = 250
		}
		s.walks[id] = mapWalk{seq: token.GetAnimSeq(), route: route, stepMS: step, startMS: nowMS, dash: token.GetAnim() == "dash"}
	}
}

// walkRoute is the path with its start cell in front: the engine's paths
// leave out the cell the token stood on.
func walkRoute(from *df.Cell, path []*df.Cell, to *df.Cell) []*df.Cell {
	route := make([]*df.Cell, 0, len(path)+1)
	if from != nil && len(path) > 0 && !sameCell(from, path[0].GetC(), path[0].GetR()) && adjacentCells(from, path[0]) {
		route = append(route, cloneMessage(from))
	}
	for _, cell := range path {
		route = append(route, cloneMessage(cell))
	}
	if to != nil && (len(route) == 0 || !sameCell(route[len(route)-1], to.GetC(), to.GetR())) {
		return nil
	}
	return route
}

func adjacentCells(a, b *df.Cell) bool {
	dc, dr := a.GetC()-b.GetC(), a.GetR()-b.GetR()
	return dc >= -1 && dc <= 1 && dr >= -1 && dr <= 1
}

func reachPath(grid *df.MiniGrid, cell *df.Cell) (*df.ReachPath, bool) {
	if cell == nil {
		return nil, false
	}
	for _, entry := range grid.GetPaths() {
		if sameCell(entry.GetCell(), cell.GetC(), cell.GetR()) {
			return entry, true
		}
	}
	return nil, false
}

// MapLayout projects the current MiniGrid into the displayed map at nowMS.
func (m *CombatModel) MapLayout(nowMS float64) CombatMapLayout {
	if m == nil {
		return CombatMapLayout{}
	}
	layout := buildMapLayout(m.state.MiniGrid, m.mapState, m.state.CanAct && m.mapState.open, nowMS)
	layout.MoveLeft = m.state.MoveLeftCells
	return layout
}

// SetMapClock installs the monotonic millisecond clock walks are timed with.
func (m *CombatModel) SetMapClock(clock func() float64) {
	if m != nil {
		m.mapState.clock = clock
	}
}

// MapNow returns the map clock's current time in milliseconds.
func (m *CombatModel) MapNow() float64 {
	if m == nil {
		return 0
	}
	return m.mapState.nowMS()
}

func (s combatMapState) nowMS() float64 {
	if s.clock == nil {
		return 0
	}
	return s.clock()
}

func buildMapLayout(grid *df.MiniGrid, state combatMapState, interactive bool, nowMS float64) CombatMapLayout {
	if grid.GetCols() <= 0 || grid.GetRows() <= 0 {
		return CombatMapLayout{}
	}
	minC, minR, maxC, maxR := mapBounds(grid)
	cols, rows := int(maxC-minC+1), int(maxR-minR+1)
	layout := CombatMapLayout{Cols: cols, Rows: rows, Rotated: cols > rows, Interactive: interactive, CanDash: grid.GetCanDash(), Pending: state.pending, minC: minC, minR: minR}
	if layout.Rotated {
		layout.Cols, layout.Rows = rows, cols
	}
	pathSteps := map[[2]int32]int{}
	if selected, ok := reachPath(grid, state.selected); ok && interactive {
		layout.Selected, layout.Dash = cloneMessage(selected.GetCell()), selected.GetDash()
		for index, cell := range selected.GetPath() {
			pathSteps[[2]int32{cell.GetC(), cell.GetR()}] = index + 1
		}
		layout.Preview = movePreviewLabel(len(selected.GetPath()), selected.GetDash())
	}
	occupied := map[[2]int32]bool{}
	for _, token := range grid.GetTokens() {
		occupied[[2]int32{token.GetCell().GetC(), token.GetCell().GetR()}] = true
	}
	for row := minR; row <= maxR; row++ {
		for column := minC; column <= maxC; column++ {
			x, y := displayXY(layout, column, row)
			key := [2]int32{column, row}
			cell := CombatMapCell{Cell: &df.Cell{C: column, R: row}, X: x, Y: y, Walkable: containsCell(grid.GetWalkable(), column, row), Occupied: occupied[key], PathStep: pathSteps[key]}
			if interactive {
				cell.Reach = containsCell(grid.GetReachable(), column, row)
				cell.Dash = containsCell(grid.GetDashReachable(), column, row)
				cell.Selected = layout.Selected != nil && sameCell(layout.Selected, column, row)
			}
			layout.Cells = append(layout.Cells, cell)
		}
	}
	sort.SliceStable(layout.Cells, func(i, j int) bool {
		if layout.Cells[i].Y != layout.Cells[j].Y {
			return layout.Cells[i].Y < layout.Cells[j].Y
		}
		return layout.Cells[i].X < layout.Cells[j].X
	})
	for _, token := range grid.GetTokens() {
		layout.Tokens = append(layout.Tokens, mapTokenView(layout, token, state.walks[token.GetTokenId()], nowMS))
	}
	return layout
}

func mapTokenView(layout CombatMapLayout, token *df.MapToken, walk mapWalk, nowMS float64) CombatMapToken {
	x, y := displayXY(layout, token.GetCell().GetC(), token.GetCell().GetR())
	out := CombatMapToken{
		ID: token.GetTokenId(), Name: token.GetName(), Kind: token.GetKind(), Portrait: token.GetPortraitUrl(),
		X: x, Y: y, Me: token.GetMe(), Enemy: token.GetEnemy(), Down: token.GetDown(), Active: token.GetActive(),
	}
	if walk.seq == 0 || walk.seq != token.GetAnimSeq() || len(walk.route) < 2 {
		return out
	}
	duration := walk.stepMS * (len(walk.route) - 1)
	elapsed := int(nowMS - walk.startMS)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= duration {
		return out
	}
	name := "dfcm-" + cssIdent(token.GetTokenId()) + "-" + strconv.FormatUint(walk.seq, 10)
	out.Walk = &CombatMapWalk{Name: name, Keyframes: walkKeyframes(layout, name, walk.route, x, y), DurationMS: duration, ElapsedMS: elapsed, Dash: walk.dash}
	return out
}

// walkKeyframes moves a one-cell token box along route; translate
// percentages are of the token's own size, so 100% is one cell.
func walkKeyframes(layout CombatMapLayout, name string, route []*df.Cell, destX, destY int) string {
	var text strings.Builder
	text.WriteString("@keyframes " + name + "{")
	last := len(route) - 1
	for index, cell := range route {
		x, y := displayXY(layout, cell.GetC(), cell.GetR())
		percent := strconv.FormatFloat(float64(index)*100/float64(last), 'f', 2, 64)
		text.WriteString(percent + "%{transform:translate(" + strconv.Itoa((x-destX)*100) + "%," + strconv.Itoa((y-destY)*100) + "%)}")
	}
	text.WriteString("}")
	return text.String()
}

// displayXY maps a grid cell to its displayed column and row.
func displayXY(layout CombatMapLayout, column, row int32) (int, int) {
	column, row = column-layout.minC, row-layout.minR
	if layout.Rotated {
		return layout.Cols - 1 - int(row), int(column)
	}
	return int(column), int(row)
}

// movePreviewLabel is the commit bar text for a picked cell.
func movePreviewLabel(cells int, dash bool) string {
	unit := " cells"
	if cells == 1 {
		unit = " cell"
	}
	if dash {
		return "Dash · " + strconv.Itoa(cells) + unit + " (ends your turn)"
	}
	return "Move · " + strconv.Itoa(cells) + unit
}

// mapHeadline summarises the movement left above the map.
func mapHeadline(moveLeft int32, canDash bool) string {
	text := strconv.Itoa(int(moveLeft)) + " of 6 cells left"
	if moveLeft == 1 {
		text = "1 of 6 cells left"
	}
	if canDash {
		return text + " · Dash to go further"
	}
	return text
}

func cssIdent(value string) string {
	var out strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// mapBounds is the smallest grid window holding every walkable cell and
// token; a grid with neither keeps its full size.
func mapBounds(grid *df.MiniGrid) (int32, int32, int32, int32) {
	minC, minR, maxC, maxR := grid.GetCols(), grid.GetRows(), int32(-1), int32(-1)
	grow := func(cell *df.Cell) {
		if cell == nil || cell.GetC() < 0 || cell.GetR() < 0 || cell.GetC() >= grid.GetCols() || cell.GetR() >= grid.GetRows() {
			return
		}
		minC, maxC = min(minC, cell.GetC()), max(maxC, cell.GetC())
		minR, maxR = min(minR, cell.GetR()), max(maxR, cell.GetR())
	}
	for _, cell := range grid.GetWalkable() {
		grow(cell)
	}
	for _, token := range grid.GetTokens() {
		grow(token.GetCell())
	}
	if maxC < 0 {
		return 0, 0, grid.GetCols() - 1, grid.GetRows() - 1
	}
	return minC, minR, maxC, maxR
}
