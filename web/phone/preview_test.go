package phone

import "testing"

func TestPreviews_CoverEveryPhoneState(t *testing.T) {
	want := map[string]ScreenKind{
		"join": ScreenMoves, "creation-pick": ScreenCreate, "creation-rolled": ScreenCreate,
		"sheet": ScreenSheet, "legal-moves": ScreenMoves, "ptt-idle": ScreenConversation,
		"ptt-recording": ScreenConversation, "ptt-sending": ScreenConversation, "typed-input": ScreenConversation,
		"dice-offered": ScreenDice, "dice-rolled": ScreenDice, "combat-my-turn": ScreenCombat,
		"combat-waiting": ScreenCombat, "down": ScreenCombat, "end": ScreenSheet,
		"combat-move": ScreenCombat, "combat-watch": ScreenCombat,
	}
	got := Previews()
	if len(got) != len(want) {
		t.Fatalf("preview count = %d, want %d", len(got), len(want))
	}
	for _, item := range got {
		t.Run(item.Name, func(t *testing.T) {
			if want[item.Name] != item.Screen {
				t.Fatalf("screen = %q, want %q", item.Screen, want[item.Name])
			}
			if item.View.Phone == nil || item.View.Phone.GetLocale() != "en" {
				t.Fatal("fixture must have an English phone view")
			}
		})
	}
}

func TestPreview_LookupAndFixtureDetails(t *testing.T) {
	item, ok := Preview("legal-moves")
	if !ok || len(item.View.Phone.GetMoves()) != 3 || item.View.Phone.GetMoves()[1].GetEnabled() || item.View.Phone.GetMoves()[1].GetReason() == "" {
		t.Fatalf("legal move fixture = %+v", item)
	}
	if _, ok := Preview("missing"); ok {
		t.Fatal("unknown preview resolved")
	}
	view, ok := PreviewSeatView("combat-my-turn")
	if !ok || !view.Phone.GetCombat().GetMyTurn() {
		t.Fatal("combat preview did not preserve turn state")
	}
}
