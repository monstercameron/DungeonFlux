package content

import "testing"

func TestCannedLines_AllSpecEntries(t *testing.T) {
	lines := CannedLines()
	if got, want := len(lines), 13; got != want {
		t.Fatalf("CannedLines() length = %d, want %d", got, want)
	}
	for _, line := range lines {
		if line.ID == "" || line.Role == "" || line.Voice == "" || line.Text == "" {
			t.Errorf("incomplete canned line: %+v", line)
		}
	}
	for _, id := range []string{CannedSlainBySeat1ID, CannedSlainBySeat2ID} {
		line, ok := CannedLineByID(id)
		if !ok {
			t.Fatalf("missing independent combat line %q", id)
		}
		if line.Role != "combat_outcomes" {
			t.Errorf("%q role = %q, want combat_outcomes", id, line.Role)
		}
	}
}

func TestCannedLineByID_NotFoundAndCopy(t *testing.T) {
	if _, ok := CannedLineByID("missing"); ok {
		t.Fatal("CannedLineByID(missing) unexpectedly found a line")
	}
	lines := CannedLines()
	lines[0].Text = "changed"
	line, ok := CannedLineByID(CannedOpeningID)
	if !ok || line.Text == "changed" {
		t.Fatal("CannedLines exposed mutable backing data")
	}
}
