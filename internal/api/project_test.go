package api

import (
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestProject_AllViewFieldsReachClientViews(t *testing.T) {
	view := sampleView()
	dm := Project(view, df.ClientKind_CLIENT_KIND_DM, 1)
	if dm.Version != 42 || dm.Phase != string(vocab.StateCombat) || dm.SpotlightSeat != "1" || !dm.Paused {
		t.Fatalf("screen metadata = %#v", dm)
	}
	if got := dm.GetDm(); got.BackgroundUrl != "bg" || len(got.Layers) != 2 || got.Narration.TextSoFar != "narration" || got.Subtitle.Text != "subtitle" || got.Callout != "callout" || len(got.BuildCards) != 1 || len(got.Preload) != 1 {
		t.Fatalf("dm scene fields = %#v", got)
	}
	if got := dm.GetDm(); got.Dice.D20 != 19 || got.Dice.Damage.Total != 8 || got.TurnTimer.RemainingMs != 500 || got.Music.TrackId != "track" {
		t.Fatalf("dm activity fields = %#v", got)
	}
	field := dm.GetDm().Battlefield
	if field.Mode != "splat" || !field.Visible || len(field.Cameras) != 1 || len(field.Grid.Walkable) != 1 || len(dm.GetDm().Tokens) != 1 || len(dm.GetDm().Highlights) != 1 || len(dm.GetDm().TurnOrder) != 1 {
		t.Fatalf("battlefield fields = %#v", field)
	}

	phone := Project(view, df.ClientKind_CLIENT_KIND_PHONE, 1).GetPhone()
	if phone.Character.Name != "Astra" || len(phone.Moves) != 1 || phone.Moves[0].Preview.Damage.Bonus != 2 || phone.TurnTimer.TotalMs != 1000 || phone.StatusText != "your turn" || phone.Combat.ContactInMs != 250 {
		t.Fatalf("phone fields = %#v", phone)
	}
	host := Project(view, df.ClientKind_CLIENT_KIND_HOST, 1).GetHost()
	if host.RunMode != "stage" || host.NextD20 != 19 || host.CombatCapRemainingMs != 900 || len(host.AssetSlots) != 1 || host.Dm == nil {
		t.Fatalf("host fields = %#v", host)
	}
}

func TestProject_NilAndUnknownValues(t *testing.T) {
	view := domain.View{Seats: []domain.SeatView{{Seat: 2, TurnTimer: domain.TimerView{}}}}
	if got := ProjectDM(view); got.Dice != nil || got.Music != nil || got.TurnTimer != nil {
		t.Fatalf("empty optional fields = %#v", got)
	}
	if got := ProjectPhone(view, 1); got.Character != nil || len(got.Moves) != 0 {
		t.Fatalf("missing seat fields = %#v", got)
	}
	if got := diceState("other"); got != df.DiceState_DICE_STATE_UNSPECIFIED {
		t.Fatalf("unknown dice state = %v", got)
	}
	if got := diceKind("other"); got != df.DiceKind_DICE_KIND_UNSPECIFIED {
		t.Fatalf("unknown dice kind = %v", got)
	}
	if ProjectHost(view).RunMode != "" || projectCharacter(nil) != nil || timerRemaining(nil) != 0 {
		t.Fatal("nil and direct projection cases were not handled")
	}
	if got := projectPhoneCombat(domain.CombatView{Tokens: []domain.TokenView{{ID: "t", Active: true, Status: "bloodied"}}}); !got.MyTurn || got.TokenId != "t" {
		t.Fatalf("phone combat token = %#v", got)
	}
}

func TestProject_EnumValues(t *testing.T) {
	for _, state := range []string{"offered", "rolling", "resolved"} {
		if diceState(state) == df.DiceState_DICE_STATE_UNSPECIFIED {
			t.Fatalf("dice state %q was not projected", state)
		}
	}
	if diceKind("check") == df.DiceKind_DICE_KIND_UNSPECIFIED {
		t.Fatal("check kind was not projected")
	}
}

func sampleView() domain.View {
	return domain.View{
		Version: 42, At: 3 * time.Second, Path: vocab.StateCombat, Paused: true, Spotlight: 1, RunMode: vocab.RunStage, NextD20: 19,
		Seats:       []domain.SeatView{{Seat: 1, Build: &domain.BuildCard{Name: "Astra", Class: "Rogue", Portrait: "portrait", PlayerNumber: 1}, Character: &domain.Character{Name: "Astra", Class: "Rogue", Hook: "river", Portrait: "portrait", PersuasionModifier: 4}, Moves: []domain.MoveView{{ID: vocab.MoveAttack, Label: "Attack", Enabled: true, TargetID: "thrall", Cell: domain.Cell{C: 2, R: 3}, Options: []domain.OptionView{{ID: "x", Label: "X"}}, Preview: &domain.MovePreview{Modifier: 4, VS: 12, PSuccess: .65, Damage: domain.DamageView{Dice: "1d6", Bonus: 2}}}}, TurnTimer: domain.TimerView{Name: "turn", RemainingMS: 500, TotalMS: 1000}, StatusText: "your turn"}},
		Scene:       domain.SceneView{BackgroundURL: "bg", Layers: []string{"one", "two"}, Narration: "narration", Subtitle: "subtitle"},
		Dice:        &domain.DiceView{State: "resolved", D20: 19, Modifier: 4, DC: 12, Outcome: "hit", Kind: "attack", VSLLabel: "vs AC", Crit: true, Damage: &domain.DamageView{Dice: "1d6", Faces: []int{6}, Bonus: 2, Total: 8, Type: "piercing"}},
		Battlefield: &domain.BattlefieldView{Mode: "splat", Visible: true, SceneURL: "scene", LiteURL: "lite", Grid: domain.Grid{Origin: [2]float64{1, 2}, CellM: 1.5, Cols: 2, Rows: 2, Walkable: []bool{true, false, false, false}}, Cameras: map[string]domain.CameraDef{"hero": {}}, Camera: domain.CameraView{Preset: "hero", FocusTokenID: "hero", Seq: 7}, Tokens: []domain.TokenView{{ID: "hero", Name: "Astra", Cell: domain.Cell{C: 1, R: 1}, Portrait: "portrait", Status: "ready", HP: 9, HPMax: 10, Active: true}}, Highlights: []domain.HighlightView{{Kind: "move", Cells: []domain.Cell{{C: 2, R: 2}}}}, TurnOrder: []domain.TurnEntry{{TokenID: "hero", Name: "Astra", Portrait: "portrait", HP: 9, HPMax: 10, Active: true}}, Round: 2},
		Combat:      &domain.CombatView{Contact: domain.TimerView{RemainingMS: 250}, Cap: domain.TimerView{RemainingMS: 900}, Banner: "combat"},
		Preload:     []string{"asset"}, Callout: "callout", Music: domain.MusicView{TrackID: "track", URL: "music", LoopStartMS: 10, LoopEndMS: 20, BPM: 90, Level: .8, Duck: .2, Cue: "cue"}, Slots: []domain.SlotView{{Name: "portrait", State: "ready", Asset: "portrait"}},
	}
}
