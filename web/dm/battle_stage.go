package dm

import (
	"encoding/json"
	"fmt"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/splat"
)

// BattleStageModel is the browser-ready PlayCanvas scene and combat snapshot.
type BattleStageModel struct {
	Enabled bool
	Init    splat.Init
	Scene   splat.Scene
	Effects splat.Effects
	HP      map[string]int32
}

// BattleStageFromView maps a server snapshot to the TV battle stage.
func BattleStageFromView(view *dungeonfluxv1.DMView, sequence uint64) BattleStageModel {
	if view == nil {
		return BattleStageModel{}
	}
	battlefield := view.GetBattlefield()
	if battlefield != nil && strings.EqualFold(battlefield.GetMode(), "FLAT") {
		return BattleStageModel{}
	}
	if sequence == 0 {
		sequence = 1
	}
	grid := splat.WoodedPathGrid()
	transform := splat.Transform{Scale: 0.4, Translate: [3]float64{0, 9.2, 0}}
	sceneURL, liteURL, visible := splat.WoodedPathSceneURL, splat.WoodedPathSceneURL, true
	if battlefield != nil {
		if battlefield.GetSceneUrl() != "" {
			sceneURL = battlefield.GetSceneUrl()
		}
		if battlefield.GetLiteUrl() != "" {
			liteURL = battlefield.GetLiteUrl()
		}
		visible = battlefield.GetVisible()
		if battlefield.GetTransform() != "" {
			_ = json.Unmarshal([]byte(battlefield.GetTransform()), &transform)
		}
		if mapped := splatGrid(battlefield.GetGrid(), sceneURL); mapped.Cols > 0 && mapped.Rows > 0 {
			grid = mapped
		}
	}
	activeID := ""
	for index, token := range view.GetTokens() {
		if token != nil && token.GetActive() {
			activeID = stageTokenID(token, index)
			break
		}
	}
	camera := splat.CameraCommand{Preset: "COMBAT_EST", FocusTokenID: activeID, Seq: sequence}
	follow := activeID != ""
	camera.Follow = &follow
	if battlefield != nil && battlefield.GetCamera() != nil {
		command := battlefield.GetCamera()
		camera.Preset = command.GetPreset()
		camera.FocusTokenID = command.GetFocusTokenId()
		follow = command.GetFollow()
		camera.Follow = &follow
		camera.DurationMS = float64(command.GetDurationMs())
	}
	tokens := stageTokens(view.GetTokens(), view.GetBuildCards())
	tokens = supportedStageTokens(tokens, grid)
	highlights := stageHighlights(view.GetHighlights())
	hp := make(map[string]int32, len(view.GetTokens()))
	for index, token := range view.GetTokens() {
		if token != nil {
			hp[stageTokenID(token, index)] = token.GetHp()
		}
	}
	effects := splat.Effects{Seq: sequence}
	if battlefield != nil {
		if shake := battlefield.GetShake(); shake != nil {
			effects.Shake = &splat.Shake{AmplitudePX: float64(shake.GetAmplitudePx()), DurationMS: float64(shake.GetDurationMs())}
		}
	} else if shake := view.GetShake(); shake != nil {
		effects.Shake = &splat.Shake{AmplitudePX: float64(shake.GetAmplitudePx()), DurationMS: float64(shake.GetDurationMs())}
	}
	return BattleStageModel{Enabled: true, Init: splat.Init{CanvasID: "df-combat-splat", SceneURL: sceneURL, LiteURL: liteURL, Transform: transform, Grid: grid, Device: "webgl2"}, Scene: splat.Scene{Seq: sequence, Visible: visible, Tokens: tokens, Highlights: highlights, Camera: camera}, Effects: effects, HP: hp}
}

func supportedStageTokens(tokens []splat.Token, grid splat.Grid) []splat.Token {
	result := make([]splat.Token, 0, len(tokens))
	for _, token := range tokens {
		if splat.IsWalkable(grid, token.Cell) {
			result = append(result, token)
		}
	}
	return result
}

func splatGrid(grid *dungeonfluxv1.Grid, sceneURL string) splat.Grid {
	if grid == nil || grid.GetCols() <= 0 || grid.GetRows() <= 0 {
		return splat.Grid{}
	}
	cells := make([]splat.Cell, 0, len(grid.GetWalkable()))
	for _, cell := range grid.GetWalkable() {
		if cell != nil {
			cells = append(cells, splat.Cell{int(cell.GetC()), int(cell.GetR())})
		}
	}
	result := splat.GridFromSupportedCells(int(grid.GetCols()), int(grid.GetRows()), cells)
	result.CellM = float64(grid.GetCellM())
	if origin := grid.GetOrigin(); origin != nil {
		result.Origin = [2]float64{float64(origin.GetC()), float64(origin.GetR())}
	}
	if sceneURL == splat.WoodedPathSceneURL && result.Cols == 16 && result.Rows == 10 {
		profile := splat.WoodedPathGrid()
		result.Origin, result.CellM = profile.Origin, profile.CellM
	}
	return result
}

func stageTokens(tokens []*dungeonfluxv1.Token, cards []*dungeonfluxv1.BuildCard) []splat.Token {
	result := make([]splat.Token, 0, len(tokens))
	for index, token := range tokens {
		if token == nil || token.GetCell() == nil {
			continue
		}
		kind := token.GetKind()
		if kind == "" {
			kind = stageTokenKind(token, cards)
		}
		anim := token.GetAnim()
		if anim == "" {
			anim = "idle"
		}
		result = append(result, splat.Token{ID: stageTokenID(token, index), Kind: kind, Name: token.GetName(), Cell: splat.Cell{int(token.GetCell().GetC()), int(token.GetCell().GetR())}, Path: stageCells(token.GetPath()), HeightM: heroHeightM, Portrait: token.GetPortraitUrl(), Anim: anim, AnimSeq: token.GetAnimSeq(), StepMS: int64(token.GetStepMs()), Clips: copyClips(token.GetClips()), Statuses: append([]string(nil), token.GetStatuses()...)})
		if kind == "thrall" {
			result[len(result)-1].HeightM = thrallHeightM
		}
	}
	return result
}

// Token heights on the TV are larger than life on purpose: at 1.8 m on 1.5 m
// cells the heroes read as specks from the tactical camera.
const (
	heroHeightM   = 2.7
	thrallHeightM = 2.5
)

func stageTokenID(token *dungeonfluxv1.Token, index int) string {
	if strings.Contains(strings.ToLower(token.GetName()), "thrall") {
		return "thrall"
	}
	if strings.HasPrefix(strings.ToLower(token.GetTokenId()), "pc-") {
		return token.GetTokenId()
	}
	if index < 2 {
		return fmt.Sprintf("pc-%d", index+1)
	}
	return token.GetTokenId()
}

func stageTokenKind(token *dungeonfluxv1.Token, cards []*dungeonfluxv1.BuildCard) string {
	if strings.Contains(strings.ToLower(token.GetName()), "thrall") {
		return "thrall"
	}
	className := tokenClass(token, cards)
	if className == "" {
		return "pc"
	}
	return "pc-" + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(className), " ", "-"))
}

func stageHighlights(highlights []*dungeonfluxv1.Highlight) []splat.Highlight {
	result := make([]splat.Highlight, 0, len(highlights))
	for _, highlight := range highlights {
		if highlight == nil {
			continue
		}
		cells := stageCells(highlight.GetCells())
		if len(cells) == 0 && highlight.GetCell() != nil {
			cells = []splat.Cell{{int(highlight.GetCell().GetC()), int(highlight.GetCell().GetR())}}
		}
		if len(cells) > 0 {
			result = append(result, splat.Highlight{Kind: highlight.GetKind(), Cells: cells})
		}
	}
	return result
}

func stageCells(cells []*dungeonfluxv1.Cell) []splat.Cell {
	if len(cells) == 0 {
		return nil
	}
	result := make([]splat.Cell, 0, len(cells))
	for _, cell := range cells {
		if cell != nil {
			result = append(result, splat.Cell{int(cell.GetC()), int(cell.GetR())})
		}
	}
	return result
}

func copyClips(clips map[string]string) map[string]string {
	if len(clips) == 0 {
		return nil
	}
	result := make(map[string]string, len(clips))
	for name, clip := range clips {
		result[name] = clip
	}
	return result
}
