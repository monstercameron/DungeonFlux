package wire

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/config"
)

// newBudget constructs the run ledger owned by the composition root. Vendor
// caps are intentionally left open; the configured hard cap remains the
// authoritative ceiling until per-vendor pricing is supplied by config.
func newBudget(cfg config.Config) (*budget.Ledger, error) {
	ledger, err := budget.NewLedger(nil, cfg.Budget.HardUSD)
	if err != nil {
		return nil, fmt.Errorf("wire: create budget ledger: %w", err)
	}
	return ledger, nil
}
