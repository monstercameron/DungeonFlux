package dice

import "testing"

func TestRollDeterministicAndCounters(t *testing.T) {
	a, b := New([]byte("seed")), New([]byte("seed"))
	for i := 0; i < 100; i++ {
		ra, err := a.Roll(20)
		if err != nil {
			t.Fatal(err)
		}
		rb, err := b.Roll(20)
		if err != nil {
			t.Fatal(err)
		}
		if ra != rb || ra.Face < 1 || ra.Face > 20 {
			t.Fatalf("draw %d mismatch: %#v %#v", i, ra, rb)
		}
	}
	if a.Counter() != 100 {
		t.Fatalf("counter=%d", a.Counter())
	}
}

func TestRollD20ForceConsumedOnce(t *testing.T) {
	r := New(make([]byte, 32))
	if err := r.ForceD20(17); err != nil {
		t.Fatal(err)
	}
	faces, kept, source, err := r.RollD20(1)
	if err != nil || source != "FORCED" || kept.Face != 17 || len(faces) != 2 || faces[0] != 17 || r.Counter() != 2 {
		t.Fatalf("forced result: %#v %v %s counter=%d", faces, err, source, r.Counter())
	}
	_, next, source, err := r.RollD20(0)
	if err != nil || source != "RNG" || next.Face == 17 && r.Counter() == 0 {
		t.Fatalf("force was not consumed: %#v %s", next, source)
	}
}

func TestRollValidationAndDistribution(t *testing.T) {
	r := New([]byte("distribution"))
	if _, err := r.Roll(0); err == nil {
		t.Fatal("expected invalid sides")
	}
	if err := r.ForceD20(21); err == nil {
		t.Fatal("expected invalid force")
	}
	counts := make([]int, 5)
	for i := 0; i < 5000; i++ {
		draw, err := r.Roll(5)
		if err != nil {
			t.Fatal(err)
		}
		counts[draw.Face-1]++
	}
	for face, count := range counts {
		if count < 850 || count > 1150 {
			t.Fatalf("face %d count=%d", face+1, count)
		}
	}
}
