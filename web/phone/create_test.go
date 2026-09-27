package phone

import (
	"context"
	"errors"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCreationModel_SelectRollLockAndProject(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCreationModel(fake, " seat-1 ", 1)
	if err := model.SelectSpecies(" ELF "); err != nil {
		t.Fatal(err)
	}
	if err := model.SelectGender("female"); err != nil {
		t.Fatal(err)
	}
	if err := model.SelectClass("rogue"); err != nil {
		t.Fatal(err)
	}
	model.ApplyAct(<-model.RollHero(context.Background()))
	if fake.request.GetMoveId() != "roll_hero" || fake.request.GetSeatToken() != "seat-1" {
		t.Fatalf("request = %+v", fake.request)
	}
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra", ClassName: "Rogue", PortraitUrl: "asset"}}}})
	if got := model.Snapshot(); got.Build.GetName() != "Astra" || got.Class != "rogue" || got.Phase != CreationRolling {
		t.Fatalf("snapshot = %+v", got)
	}
	model.ApplyAct(<-model.Lock(context.Background()))
	if model.Snapshot().Phase != CreationLocked || fake.request.GetMoveId() != "ready" {
		t.Fatalf("locked snapshot = %+v", model.Snapshot())
	}
}

func TestCreationModel_RollSendsAllPicksInOrder(t *testing.T) {
	fake := &recordingActFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCreationModel(fake, "seat", 1)
	_ = model.SelectSpecies("human")
	_ = model.SelectGender("female")
	_ = model.SelectClass("paladin")
	model.ApplyAct(<-model.RollHero(context.Background()))
	if len(fake.requests) != 4 {
		t.Fatalf("requests = %d, want 4", len(fake.requests))
	}
	for i, want := range []struct{ move, arg string }{{"species", "human"}, {"gender", "female"}, {"class", "paladin"}, {"roll_hero", ""}} {
		if got := fake.requests[i]; got.GetMoveId() != want.move || got.GetArg() != want.arg {
			t.Fatalf("request %d = %v, want %s/%s", i, got, want.move, want.arg)
		}
	}
}

func TestCreationModel_ProjectsCharacterFieldsAndLock(t *testing.T) {
	model := NewCreationModel(&actFake{}, "seat", 1)
	state := &df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		StatusText: "Your hero is ready to lock in",
		Character:  &df.Character{Name: "Rook", ClassName: "rogue", Species: "human", Gender: "female", Locked: false, Build: &df.CharacterBuild{Hp: 11, HpMax: 11, Ac: 14}},
	}}}
	got := model.ApplyScreenState(state)
	if got.Class != "rogue" || got.Species != "human" || got.Gender != "female" || got.StatusText != "Your hero is ready to lock in" || got.Phase != CreationRolling {
		t.Fatalf("rolled snapshot = %+v", got)
	}
	state.GetPhone().GetCharacter().Locked = true
	got = model.ApplyScreenState(state)
	if got.Phase != CreationLocked {
		t.Fatalf("locked snapshot = %+v", got)
	}
}

func TestCreationClasses_AreTwelveStableCopies(t *testing.T) {
	classes := CreationClasses()
	if len(classes) != 12 || classes[0].ID != "barbarian" || classes[len(classes)-1].ID != "wizard" {
		t.Fatalf("classes = %+v", classes)
	}
	for _, class := range classes {
		if class.Label == "" || class.Role == "" || class.Crest == "" {
			t.Fatalf("incomplete class = %+v", class)
		}
		if err := (&CreationModel{}).SelectClass(class.ID); err != nil {
			t.Fatalf("class %q rejected: %v", class.ID, err)
		}
	}
	classes[0].ID = "changed"
	if CreationClasses()[0].ID != "barbarian" {
		t.Fatal("class result aliases package state")
	}
	if got := CreationClassesForLocale("es-MX")[0].Label; got != "Bárbaro" {
		t.Fatalf("Spanish class = %q", got)
	}
}

func TestCreationModel_RejectsInvalidSelectionsAndMissingChoices(t *testing.T) {
	model := NewCreationModel(&actFake{}, "token", 1)
	for _, test := range []struct {
		name string
		call func() error
	}{
		{"species", func() error { return model.SelectSpecies("kobold") }},
		{"gender", func() error { return model.SelectGender("unknown") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); err == nil {
				t.Fatal("invalid value accepted")
			}
		})
	}
	if err := (<-model.RollHero(context.Background())).Err; err == nil || err.Error() != "species, gender, and class are required" {
		t.Fatalf("missing choices error = %v", err)
	}
}

func TestCreationModel_RecordsRPCFailures(t *testing.T) {
	fake := &actFake{result: ActResult{Err: errors.New("offline")}}
	model := NewCreationModel(fake, "token", 2)
	_ = model.SelectSpecies("human")
	_ = model.SelectGender("nonbinary")
	_ = model.SelectClass("wizard")
	got := model.ApplyAct(<-model.RollHero(context.Background()))
	if got.Phase != CreationFailed || got.Error != "offline" {
		t.Fatalf("failure = %+v", got)
	}
	if NewCreationModel(nil, "", 0).Snapshot().Phase != CreationPicking {
		t.Fatal("new model should start in picking")
	}
}

func TestCreationOptions_AreCopiesWithStableLegalValues(t *testing.T) {
	species := CreationSpecies()
	genders := CreationGenders()
	if len(species) != 9 || len(genders) != 3 {
		t.Fatalf("option counts = %d/%d", len(species), len(genders))
	}
	if species[0].ID != "human" || species[len(species)-1].ID != "goliath" || genders[2].ID != "nonbinary" {
		t.Fatalf("unexpected option order: %+v / %+v", species, genders)
	}
	species[0].ID = "changed"
	if CreationSpecies()[0].ID != "human" {
		t.Fatal("species result aliases package state")
	}
}

func TestCreationModel_RejectsServerResponseAndPreservesSelection(t *testing.T) {
	model := NewCreationModel(&actFake{result: ActResult{Value: &df.ActResponse{Accepted: false, Reason: "seat is locked"}}}, "token", 1)
	if err := model.SelectSpecies("elf"); err != nil {
		t.Fatal(err)
	}
	if err := model.SelectGender("male"); err != nil {
		t.Fatal(err)
	}
	if err := model.SelectClass("bard"); err != nil {
		t.Fatal(err)
	}
	got := model.ApplyAct(<-model.RollHero(context.Background()))
	if got.Phase != CreationFailed || got.Error != "seat is locked" || got.Species != "elf" || got.Gender != "male" {
		t.Fatalf("rejection snapshot = %+v", got)
	}
}

type actFake struct {
	request *df.ActRequest
	result  ActResult
}

type recordingActFake struct {
	requests []*df.ActRequest
	result   ActResult
}

func (f *recordingActFake) Act(_ context.Context, request *df.ActRequest) <-chan ActResult {
	f.requests = append(f.requests, request)
	out := make(chan ActResult, 1)
	out <- f.result
	return out
}

func (f *actFake) Act(_ context.Context, request *df.ActRequest) <-chan ActResult {
	f.request = request
	out := make(chan ActResult, 1)
	out <- f.result
	return out
}
