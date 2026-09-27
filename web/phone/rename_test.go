package phone

import (
	"context"
	"strings"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCreationModel_BeginRenameSeedsDraftFromCurrentName(t *testing.T) {
	model := NewCreationModel(&actFake{}, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{
		Character: &df.Character{Name: "Astra", ClassName: "Rogue"},
	}}})
	got := model.BeginRename()
	if !got.Renaming || got.RenameDraft != "Astra" || got.RenameError != "" {
		t.Fatalf("BeginRename snapshot = %+v", got)
	}
}

func TestCreationModel_CancelRenameClosesEditorWithoutSending(t *testing.T) {
	fake := &actFake{}
	model := NewCreationModel(fake, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft("Someone Else")
	got := model.CancelRename()
	if got.Renaming || got.RenameDraft != "" || got.RenameError != "" {
		t.Fatalf("CancelRename snapshot = %+v", got)
	}
	if fake.request != nil {
		t.Fatalf("cancel should not send an Act, got %+v", fake.request)
	}
}

func TestCreationModel_SubmitRename_TrimsAndSends(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCreationModel(fake, "seat-9", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft("  Sable Wren  ")
	outcome := <-model.SubmitRename(context.Background())
	if outcome.Err != nil {
		t.Fatalf("SubmitRename error: %v", outcome.Err)
	}
	if fake.request.GetMoveId() != "rename" || fake.request.GetArg() != "Sable Wren" || fake.request.GetSeatToken() != "seat-9" {
		t.Fatalf("request = %+v", fake.request)
	}
	got := model.Snapshot()
	if got.Renaming || got.RenameDraft != "" || got.RenameError != "" {
		t.Fatalf("snapshot after accepted rename = %+v", got)
	}
}

func TestCreationModel_SubmitRename_EmptyDraftRejectedLocallyWithoutSending(t *testing.T) {
	fake := &actFake{}
	model := NewCreationModel(fake, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft("   ")
	<-model.SubmitRename(context.Background())
	if fake.request != nil {
		t.Fatalf("an empty draft should not reach the server, got %+v", fake.request)
	}
	got := model.Snapshot()
	if !got.Renaming || got.RenameError == "" {
		t.Fatalf("snapshot after empty submit = %+v", got)
	}
}

func TestCreationModel_SubmitRename_TooLongRejectedLocally(t *testing.T) {
	fake := &actFake{}
	model := NewCreationModel(fake, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft(strings.Repeat("a", heroNameMaxLength+1))
	<-model.SubmitRename(context.Background())
	if fake.request != nil {
		t.Fatalf("an overlong draft should not reach the server, got %+v", fake.request)
	}
	if model.Snapshot().RenameError == "" {
		t.Fatal("expected a RenameError for an overlong draft")
	}
}

func TestCreationModel_SubmitRename_StripsMarkupAndControlCharacters(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: true}}}
	model := NewCreationModel(fake, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft("<b>Sa\u0007ble</b> {Wren}")
	<-model.SubmitRename(context.Background())
	if strings.ContainsAny(fake.request.GetArg(), "<>{}") {
		t.Fatalf("sent arg still has markup: %q", fake.request.GetArg())
	}
}

func TestCreationModel_SubmitRename_ServerRejectionSurfacesReason(t *testing.T) {
	fake := &actFake{result: ActResult{Value: &df.ActResponse{Accepted: false, Reason: "seat is locked"}}}
	model := NewCreationModel(fake, "seat", 1)
	model.ApplyScreenState(&df.ScreenState{Phase: "creation", View: &df.ScreenState_Phone{Phone: &df.PhoneView{Character: &df.Character{Name: "Astra"}}}})
	model.BeginRename()
	model.SetRenameDraft("New Name")
	<-model.SubmitRename(context.Background())
	got := model.Snapshot()
	if !got.Renaming || got.RenameError != "seat is locked" {
		t.Fatalf("snapshot after server rejection = %+v", got)
	}
}

func TestCreationModel_SubmitRename_WithoutClientFails(t *testing.T) {
	model := NewCreationModel(nil, "seat", 1)
	outcome := <-model.SubmitRename(context.Background())
	if outcome.Err == nil {
		t.Fatal("expected an error when the creation client is unavailable")
	}
}

func TestNilCreationModel_RenameMethodsAreSafe(t *testing.T) {
	var model *CreationModel
	if got := model.BeginRename(); got.Phase != CreationFailed {
		t.Fatalf("BeginRename on nil model = %+v", got)
	}
	if got := model.SetRenameDraft("x"); got.Phase != CreationFailed {
		t.Fatalf("SetRenameDraft on nil model = %+v", got)
	}
	if got := model.CancelRename(); got.Phase != CreationFailed {
		t.Fatalf("CancelRename on nil model = %+v", got)
	}
}

func TestSanitizeHeroNameDraft(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"trims", "  Wren  ", "Wren", false},
		{"empty", "", "", true},
		{"only_control", "\u0000\u001f", "", true},
		{"strips_markup", "<Wren>", "Wren", false},
		{"over_length", strings.Repeat("a", heroNameMaxLength+1), "", true},
		{"max_length_ok", strings.Repeat("a", heroNameMaxLength), strings.Repeat("a", heroNameMaxLength), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := sanitizeHeroNameDraft(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("sanitizeHeroNameDraft(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("sanitizeHeroNameDraft(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("sanitizeHeroNameDraft(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
