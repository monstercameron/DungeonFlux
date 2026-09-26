package budget

import "github.com/monstercameron/DungeonFlux/internal/vocab"

// Price describes a vendor's unit prices in USD. Token prices are per million
// tokens; UnitUSD is for seconds, images, or other caller-defined units.
type Price struct {
	Vendor                   vocab.VendorName
	Model                    string
	InputUSDPerMillion       float64
	CachedInputUSDPerMillion float64
	OutputUSDPerMillion      float64
	UnitUSD                  float64
}

// Usage contains the billable quantities for one vendor call.
type Usage struct {
	InputTokens       int
	CachedInputTokens int
	OutputTokens      int
	Units             float64
}

// Estimate returns the non-negative estimated cost for the supplied usage.
func (p Price) Estimate(u Usage) float64 {
	input := u.InputTokens - u.CachedInputTokens
	if input < 0 {
		input = 0
	}
	return float64(input)*p.InputUSDPerMillion/1_000_000 +
		float64(u.CachedInputTokens)*p.CachedInputUSDPerMillion/1_000_000 +
		float64(u.OutputTokens)*p.OutputUSDPerMillion/1_000_000 +
		u.Units*p.UnitUSD
}
