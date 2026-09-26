package dm

import (
	"math"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// CombatPoint is a pixel coordinate in the FLAT battlefield image.
type CombatPoint struct {
	X float32
	Y float32
}

// CombatSegment is one projected edge of a walkable grid cell.
type CombatSegment struct {
	From CombatPoint
	To   CombatPoint
}

// CombatToken is the browser-ready position and status of one combatant.
type CombatToken struct {
	ID         string
	Name       string
	Portrait   string
	X          float32
	Y          float32
	HP         int32
	HPMax      int32
	Active     bool
	Statuses   []string
	CellColumn int32
	CellRow    int32
}

// CombatTurn is one entry in the fixed, visible combat initiative order.
type CombatTurn struct {
	ID       string
	Name     string
	Portrait string
	HP       int32
	HPMax    int32
	Active   bool
	Done     bool
}

// CombatModel contains the FLAT battlefield image, projected grid, and tokens.
type CombatModel struct {
	ImageURL   string
	Visible    bool
	Segments   []CombatSegment
	Highlights []CombatPoint
	Tokens     []CombatToken
	TurnOrder  []CombatTurn
	Round      int32
	Banner     string
	Timer      TimerView
}

// CombatModelFromView projects a DM combat view into browser-owned data.
// FLAT is intentionally the only mode rendered by this package.
func CombatModelFromView(view *dungeonfluxv1.DMView) CombatModel {
	if view == nil {
		return CombatModel{}
	}
	model := CombatModel{
		Timer:     TimerViewFromProto(view.GetTurnTimer()),
		Round:     view.GetRound(),
		Banner:    view.GetCombatBanner(),
		TurnOrder: projectedTurnOrder(view.GetTurnOrder()),
	}
	battlefield := view.GetBattlefield()
	if battlefield == nil {
		return model
	}
	model.Visible = battlefield.GetVisible()
	if flat := flatBattlefield(battlefield); flat != nil {
		model.ImageURL = flat.GetImageUrl()
		model.Segments = projectedGrid(battlefield.GetGrid(), flat.GetFloorQuadPx())
		model.Tokens = projectedTokens(view.GetTokens(), battlefield.GetGrid(), flat.GetFloorQuadPx())
		model.Highlights = projectedHighlights(view.GetHighlights(), battlefield.GetGrid(), flat.GetFloorQuadPx())
	}
	return model
}

func projectedTurnOrder(entries []*dungeonfluxv1.TurnOrderEntry) []CombatTurn {
	result := make([]CombatTurn, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		result = append(result, CombatTurn{ID: entry.GetTokenId(), Name: entry.GetName(), Portrait: entry.GetPortraitUrl(), HP: entry.GetHp(), HPMax: entry.GetHpMax(), Active: entry.GetActive(), Done: entry.GetDone()})
	}
	return result
}

func projectedHighlights(highlights []*dungeonfluxv1.Highlight, grid *dungeonfluxv1.Grid, quad []float32) []CombatPoint {
	if grid == nil || len(quad) < 8 {
		return nil
	}
	h, ok := newHomography(quad)
	if !ok {
		return nil
	}
	result := make([]CombatPoint, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil || highlight.GetCell() == nil {
			continue
		}
		cell := highlight.GetCell()
		if cell.GetC() < 0 || cell.GetR() < 0 || cell.GetC() >= grid.GetCols() || cell.GetR() >= grid.GetRows() {
			continue
		}
		point, valid := h.project(float32(cell.GetC())+0.5, float32(cell.GetR())+0.5, grid.GetCols(), grid.GetRows())
		if valid {
			result = append(result, point)
		}
	}
	return result
}

func flatBattlefield(battlefield *dungeonfluxv1.Battlefield) *dungeonfluxv1.FlatBattlefield {
	if battlefield == nil || !strings.EqualFold(battlefield.GetMode(), "FLAT") {
		return nil
	}
	return battlefield.GetFlat()
}

func projectedTokens(tokens []*dungeonfluxv1.Token, grid *dungeonfluxv1.Grid, quad []float32) []CombatToken {
	if grid == nil || len(quad) < 8 {
		return nil
	}
	h, ok := newHomography(quad)
	if !ok {
		return nil
	}
	result := make([]CombatToken, 0, len(tokens))
	for _, token := range tokens {
		if token == nil || token.GetCell() == nil {
			continue
		}
		cell := token.GetCell()
		point, valid := h.project(float32(cell.GetC())+0.5, float32(cell.GetR())+0.5, grid.GetCols(), grid.GetRows())
		if !valid {
			continue
		}
		result = append(result, CombatToken{ID: token.GetTokenId(), Name: token.GetName(), Portrait: token.GetPortraitUrl(), X: point.X, Y: point.Y, HP: token.GetHp(), HPMax: token.GetHpMax(), Active: token.GetActive(), Statuses: append([]string(nil), token.GetStatuses()...), CellColumn: cell.GetC(), CellRow: cell.GetR()})
	}
	return result
}

func projectedGrid(grid *dungeonfluxv1.Grid, quad []float32) []CombatSegment {
	if grid == nil || grid.GetCols() <= 0 || grid.GetRows() <= 0 || len(quad) < 8 {
		return nil
	}
	h, ok := newHomography(quad)
	if !ok {
		return nil
	}
	seen := make(map[edgeKey]struct{})
	segments := make([]CombatSegment, 0, len(grid.GetWalkable())*4)
	for _, cell := range grid.GetWalkable() {
		if cell == nil || cell.GetC() < 0 || cell.GetR() < 0 || cell.GetC() >= grid.GetCols() || cell.GetR() >= grid.GetRows() {
			continue
		}
		corners := [][2]float32{{float32(cell.GetC()), float32(cell.GetR())}, {float32(cell.GetC() + 1), float32(cell.GetR())}, {float32(cell.GetC() + 1), float32(cell.GetR() + 1)}, {float32(cell.GetC()), float32(cell.GetR() + 1)}}
		for index := range corners {
			next := (index + 1) % len(corners)
			key := makeEdgeKey(corners[index], corners[next])
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			from, fromOK := h.project(corners[index][0], corners[index][1], grid.GetCols(), grid.GetRows())
			to, toOK := h.project(corners[next][0], corners[next][1], grid.GetCols(), grid.GetRows())
			if fromOK && toOK {
				segments = append(segments, CombatSegment{From: from, To: to})
			}
		}
	}
	return segments
}

type edgeKey struct{ ax, ay, bx, by int32 }

func makeEdgeKey(from, to [2]float32) edgeKey {
	ax, ay := int32(from[0]*2), int32(from[1]*2)
	bx, by := int32(to[0]*2), int32(to[1]*2)
	if ax > bx || (ax == bx && ay > by) {
		ax, ay, bx, by = bx, by, ax, ay
	}
	return edgeKey{ax: ax, ay: ay, bx: bx, by: by}
}

type homography [8]float64

func newHomography(quad []float32) (homography, bool) {
	if len(quad) < 8 {
		return homography{}, false
	}
	// Solve the eight projective coefficients for the four image corners.
	var matrix [8][9]float64
	for index := 0; index < 4; index++ {
		u, v := []float64{0, 1, 1, 0}[index], []float64{0, 0, 1, 1}[index]
		x, y := float64(quad[index*2]), float64(quad[index*2+1])
		row := index * 2
		matrix[row] = [9]float64{u, v, 1, 0, 0, 0, -u * x, -v * x, x}
		matrix[row+1] = [9]float64{0, 0, 0, u, v, 1, -u * y, -v * y, y}
	}
	for column := 0; column < 8; column++ {
		pivot := column
		for row := column + 1; row < 8; row++ {
			if math.Abs(matrix[row][column]) > math.Abs(matrix[pivot][column]) {
				pivot = row
			}
		}
		if math.Abs(matrix[pivot][column]) < 1e-6 {
			return homography{}, false
		}
		matrix[column], matrix[pivot] = matrix[pivot], matrix[column]
		factor := matrix[column][column]
		for value := column; value < 9; value++ {
			matrix[column][value] /= factor
		}
		for row := 0; row < 8; row++ {
			if row == column {
				continue
			}
			factor = matrix[row][column]
			for value := column; value < 9; value++ {
				matrix[row][value] -= factor * matrix[column][value]
			}
		}
	}
	var result homography
	for index := range result {
		result[index] = matrix[index][8]
	}
	return result, true
}

func (h homography) project(column, row float32, cols, rows int32) (CombatPoint, bool) {
	if cols <= 0 || rows <= 0 {
		return CombatPoint{}, false
	}
	u, v := float64(column)/float64(cols), float64(row)/float64(rows)
	denominator := h[6]*u + h[7]*v + 1
	if math.Abs(denominator) < 1e-6 {
		return CombatPoint{}, false
	}
	return CombatPoint{X: float32((h[0]*u + h[1]*v + h[2]) / denominator), Y: float32((h[3]*u + h[4]*v + h[5]) / denominator)}, true
}
