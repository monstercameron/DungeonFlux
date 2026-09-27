package full

import (
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game"
	"github.com/monstercameron/DungeonFlux/internal/sim"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// TestWalkFull_SpanishPhoneEnglishDM runs the whole game through to the end
// while projecting every view twice: a Spanish phone screen for seat 1 and
// an English DM screen. Both locales must stay consistent from lobby to end.
func TestWalkFull_SpanishPhoneEnglishDM(t *testing.T) {
	engine := game.New(domain.OneShot{ID: "i18n-walk"}, []byte("i18n-walk-seed"))
	driver := sim.New(engine, sim.Script{})
	seen := map[vocab.StateID]bool{}
	proven := map[string]bool{}
	checkWalkStep(t, driver, engine, seen, proven)
	steps := []domain.Event{
		domain.Join{Seat: 1, JoinKind: "phone", Locale: "es"},
		domain.Join{Seat: 2, JoinKind: "phone", Locale: "es"},
		domain.Act{Seat: 1, Move: vocab.MoveReady},
		domain.Act{Seat: 2, Move: vocab.MoveReady},
		domain.HostCmd{Cmd: vocab.HostStart},
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "paladin"},
		domain.Act{Seat: 1, Move: vocab.MoveRollHero},
		domain.FlavorDone{Seat: 1, Flavor: domain.Flavor{Name: "A", Hook: "H"}},
		domain.PCLocked{Seat: 1},
		domain.Act{Seat: 2, Move: vocab.MoveSpecies, Arg: "elf"},
		domain.Act{Seat: 2, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 2, Move: vocab.MoveClass, Arg: "rogue"},
		domain.Act{Seat: 2, Move: vocab.MoveRollHero},
		domain.FlavorDone{Seat: 2, Flavor: domain.Flavor{Name: "B", Hook: "H"}},
		domain.PCLocked{Seat: 2},
		domain.LineDone{UtteranceID: "opening"},
		domain.Act{Seat: 1, Move: vocab.MoveTalkVell},
		domain.Act{Seat: 1, Move: vocab.MovePersuade},
		domain.TimerFired{Name: "roll_resolved"},
		domain.LineDone{UtteranceID: "reveal"},
		domain.Act{Seat: 1, Move: vocab.MoveLeave},
		domain.LineDone{UtteranceID: "hook-arrival"},
		domain.LineDone{UtteranceID: "stranger"},
		domain.LineDone{UtteranceID: "combat-outcome"},
		domain.LineDone{UtteranceID: "cliffhanger"},
	}
	for _, event := range steps {
		output := driver.Send(event)
		if output.Ack != nil && !output.Ack.Accepted {
			t.Fatalf("event %T rejected: %#v", event, output.Ack)
		}
		checkWalkStep(t, driver, engine, seen, proven)
	}
	final := engine.View()
	if final.Path != vocab.StateEnd {
		t.Fatalf("final phase = %q, want %q", final.Path, vocab.StateEnd)
	}
	for _, want := range []vocab.StateID{vocab.StateLobby, vocab.StateCreation, vocab.StateConversation, vocab.StateEnd} {
		if !seen[want] {
			t.Fatalf("walk never visited %q: %v", want, seen)
		}
	}
	if !proven["persuade-es"] {
		t.Fatal("walk never rendered the spanish persuade label")
	}
	endPhone := api.ProjectLocalized(final, df.ClientKind_CLIENT_KIND_PHONE, 1, "es")
	endDM := api.ProjectLocalized(final, df.ClientKind_CLIENT_KIND_DM, 0, "en")
	if endPhone.GetLocale() != "es" || endDM.GetLocale() != "en" {
		t.Fatalf("end locales = %q/%q", endPhone.GetLocale(), endDM.GetLocale())
	}
}

func checkWalkStep(t *testing.T, driver *sim.Simulator, engine *game.State, seen map[vocab.StateID]bool, proven map[string]bool) {
	t.Helper()
	view := engine.View()
	for i := range view.Seats {
		view.Seats[i].Moves = engine.LegalMoveViews(view.Seats[i].Seat)
	}
	seen[view.Path] = true
	phone := api.ProjectLocalized(view, df.ClientKind_CLIENT_KIND_PHONE, 1, "es")
	dm := api.ProjectLocalized(view, df.ClientKind_CLIENT_KIND_DM, 0, "en")
	checkWalkLocales(t, view.Path, phone, dm, proven)
	if _, err := driver.Advance(time.Second); err != nil {
		t.Fatal(err)
	}
}

func checkWalkLocales(t *testing.T, path vocab.StateID, phone *df.ScreenState, dm *df.ScreenState, proven map[string]bool) {
	t.Helper()
	if phone.GetLocale() != "es" {
		t.Fatalf("phase %q: phone locale = %q, want es", path, phone.GetLocale())
	}
	if dm.GetLocale() != "en" {
		t.Fatalf("phase %q: dm locale = %q, want en", path, dm.GetLocale())
	}
	for _, move := range phone.GetPhone().GetMoves() {
		if move.GetLabelKey() == "" {
			t.Fatalf("phase %q: move %q has no label key", path, move.GetMoveId())
		}
		if move.GetMoveId() == "persuade" && move.GetLabel() == "Persuadir +4 contra CD 10" {
			proven["persuade-es"] = true
		}
		if move.GetMoveId() == "attack" && move.GetLabel() == "Atacar al ahogado" {
			proven["attack-es"] = true
		}
	}
}
