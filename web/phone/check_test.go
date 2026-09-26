package phone

import "testing"

func TestCheckPresentationFromDice_ProjectsOfferAndResult(t *testing.T) {
	tests := []struct {
		name        string
		snapshot    DiceSnapshot
		wantRoll    bool
		wantResult  string
		wantSuccess bool
	}{
		{name: "offer", snapshot: DiceSnapshot{Phase: DiceOffered, Modifier: 4, DC: 10}, wantResult: "You try to reason with Marra, appealing to her better nature."},
		{name: "success", snapshot: DiceSnapshot{Phase: DiceResolved, D20: 17, Modifier: 4, DC: 10, Outcome: "success"}, wantRoll: true, wantResult: "success", wantSuccess: true},
		{name: "failure", snapshot: DiceSnapshot{Phase: DiceResolved, D20: 2, Modifier: 4, DC: 10}, wantRoll: true, wantResult: "failure"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CheckPresentationFromDice(test.snapshot)
			if got.HasRoll != test.wantRoll {
				t.Fatalf("has roll = %v, want %v", got.HasRoll, test.wantRoll)
			}
			if test.wantResult == "success" && (!got.HasOutcome || !got.Success || got.Outcome != "Success") {
				t.Fatalf("success presentation = %+v", got)
			}
			if test.wantResult == "failure" && (!got.HasOutcome || got.Success || got.Outcome != "Failure") {
				t.Fatalf("failure presentation = %+v", got)
			}
			if test.wantResult != "success" && test.wantResult != "failure" && got.ResultText != test.wantResult {
				t.Fatalf("offer result text = %q, want %q", got.ResultText, test.wantResult)
			}
		})
	}
}

func TestCheckPresentationFromDice_UsesWireTotalAndFallbackCopy(t *testing.T) {
	got := CheckPresentationFromDice(DiceSnapshot{Phase: DiceResolved, D20: 8, Total: 15, Modifier: 4, DC: 10, Outcome: "refuse"})
	if got.Total != 15 || got.Success || got.Outcome != "Failure" || got.ResultText == "" {
		t.Fatalf("wire result = %+v", got)
	}
	if SignedModifier(4) != "+4" || SignedModifier(-2) != "-2" || SignedModifier(0) != "+0" {
		t.Fatalf("signed modifiers are not stable")
	}
}

func TestCheckPresentationFromDice_UnknownResolutionDoesNotInventOutcome(t *testing.T) {
	got := CheckPresentationFromDice(DiceSnapshot{Phase: DiceResolved, Modifier: 4, DC: 10, Outcome: "Mother Vell is deciding"})
	if got.HasOutcome || got.Success || got.Outcome != "" {
		t.Fatalf("unknown outcome = %+v", got)
	}
	if got.ResultText != "Mother Vell is deciding" {
		t.Fatalf("unknown result text = %q", got.ResultText)
	}
}
