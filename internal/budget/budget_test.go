package budget

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPrice_Estimate(t *testing.T) {
	price := Price{Vendor: vocab.VendorOpenAI, InputUSDPerMillion: 1, CachedInputUSDPerMillion: .1, OutputUSDPerMillion: 2, UnitUSD: .5}
	got := price.Estimate(Usage{InputTokens: 1000, CachedInputTokens: 400, OutputTokens: 500, Units: 2})
	if got != 1.00164 {
		t.Fatalf("cost = %f, want 1.00164", got)
	}
}

func TestLedger_ReserveSettleAndRelease(t *testing.T) {
	l, err := NewLedger(map[vocab.VendorName]float64{vocab.VendorOpenAI: 1})
	if err != nil {
		t.Fatal(err)
	}
	r, err := l.Reserve(vocab.VendorOpenAI, .4)
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Costs().ByVendor[vocab.VendorOpenAI].Reserved; got != .4 {
		t.Fatalf("reserved = %f", got)
	}
	if _, err := r.Settle(.25); err != nil {
		t.Fatal(err)
	}
	r.Release()
	s := l.Costs()
	if s.Spent != .25 || s.Reserved != 0 {
		t.Fatalf("snapshot = %+v", s)
	}
}

func TestLedger_CapReached(t *testing.T) {
	l, err := NewLedger(map[vocab.VendorName]float64{vocab.VendorGemini: .2}, .3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Reserve(vocab.VendorGemini, .21); !errors.Is(err, ErrCapReached) {
		t.Fatalf("err = %v", err)
	}
	r, err := l.Reserve(vocab.VendorGemini, .2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Settle(.2); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Reserve(vocab.VendorGemini, .01); !errors.Is(err, ErrCapReached) {
		t.Fatalf("exhausted cap err = %v", err)
	}
	overspend, err := NewLedger(map[vocab.VendorName]float64{vocab.VendorGemini: .2})
	if err != nil {
		t.Fatal(err)
	}
	r, err = overspend.Reserve(vocab.VendorGemini, .2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Settle(.25); !errors.Is(err, ErrCapReached) {
		t.Fatalf("settle err = %v", err)
	}
}

func TestNewLedger_RejectsInvalidCaps(t *testing.T) {
	cases := []struct {
		name string
		caps map[vocab.VendorName]float64
		hard []float64
	}{
		{"empty vendor", map[vocab.VendorName]float64{"": 1}, nil},
		{"negative vendor", map[vocab.VendorName]float64{vocab.VendorOpenAI: -1}, nil},
		{"negative hard", nil, []float64{-1}},
		{"too many hard", nil, []float64{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewLedger(tc.caps, tc.hard...); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
