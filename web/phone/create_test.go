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
	model.ApplyAct(<-model.RollHero(context.Background()))
	if fake.request.GetMoveId() != "roll_hero" || fake.request.GetSeatToken() != "seat-1" {
		t.Fatalf("request = %+v", fake.request)
	}
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra", ClassName: "Rogue", PortraitUrl: "asset"}}}})
	if got := model.Snapshot(); got.Build.GetName() != "Astra" || got.Phase != CreationRolling {
		t.Fatalf("snapshot = %+v", got)
	}
	model.ApplyAct(<-model.Lock(context.Background()))
	if model.Snapshot().Phase != CreationLocked || fake.request.GetMoveId() != "ready" {
		t.Fatalf("locked snapshot = %+v", model.Snapshot())
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
	if err := (<-model.RollHero(context.Background())).Err; err == nil || err.Error() != "species and gender are required" {
		t.Fatalf("missing choices error = %v", err)
	}
}

func TestCreationModel_RecordsRPCFailures(t *testing.T) {
	fake := &actFake{result: ActResult{Err: errors.New("offline")}}
	model := NewCreationModel(fake, "token", 2)
	_ = model.SelectSpecies("human")
	_ = model.SelectGender("nonbinary")
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
	got := model.ApplyAct(<-model.RollHero(context.Background()))
	if got.Phase != CreationFailed || got.Error != "seat is locked" || got.Species != "elf" || got.Gender != "male" {
		t.Fatalf("rejection snapshot = %+v", got)
	}
}

type actFake struct {
	request *df.ActRequest
	result  ActResult
}

func (f *actFake) Act(_ context.Context, request *df.ActRequest) <-chan ActResult {
	f.request = request
	out := make(chan ActResult, 1)
	out <- f.result
	return out
}
