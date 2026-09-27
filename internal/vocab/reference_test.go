package vocab

import "testing"

func TestReferenceAngles_CanonicalOrder(t *testing.T) {
	got := ReferenceAngles()
	want := []ReferenceAngle{ReferenceFront, ReferenceThreeQuarter, ReferenceSide, ReferenceBack}
	if len(got) != len(want) {
		t.Fatalf("angles=%v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("angle[%d]=%q want %q", index, got[index], want[index])
		}
	}
}
