package budget

import (
	"errors"
	"fmt"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ErrCapReached identifies a reservation rejected by a configured cap.
var ErrCapReached = errors.New("budget: cap reached")

// CapError reports the vendor and amount that could not be reserved.
type CapError struct {
	Vendor    vocab.VendorName
	Requested float64
	Remaining float64
}

// Error returns a stable, human-readable budget error.
func (e *CapError) Error() string {
	return fmt.Sprintf("budget: %s cap reached (requested %.6f, remaining %.6f)", e.Vendor, e.Requested, e.Remaining)
}

// Unwrap allows callers to use errors.Is(err, ErrCapReached).
func (e *CapError) Unwrap() error { return ErrCapReached }

// VendorCost is the settled and currently reserved spend for one vendor.
type VendorCost struct {
	Spent    float64
	Reserved float64
}

// Snapshot is a read-only copy of ledger totals.
type Snapshot struct {
	Spent    float64
	Reserved float64
	ByVendor map[vocab.VendorName]VendorCost
}

// Ledger tracks settled and in-flight spend. A zero hard cap means no total
// cap; vendor entries with a zero cap are also unlimited and omitted.
type Ledger struct {
	mu       sync.Mutex
	caps     map[vocab.VendorName]float64
	hardCap  float64
	spent    map[vocab.VendorName]float64
	reserved map[vocab.VendorName]float64
}

// NewLedger constructs a ledger. hardCap is optional; when supplied and
// positive, it caps settled plus reserved spend across all vendors.
func NewLedger(caps map[vocab.VendorName]float64, hardCap ...float64) (*Ledger, error) {
	total := 0.0
	if len(hardCap) > 1 {
		return nil, fmt.Errorf("budget: at most one hard cap is allowed")
	}
	if len(hardCap) == 1 {
		total = hardCap[0]
	}
	if total < 0 {
		return nil, fmt.Errorf("budget: hard cap is negative")
	}
	copyCaps := make(map[vocab.VendorName]float64, len(caps))
	for vendor, cap := range caps {
		if vendor == "" {
			return nil, fmt.Errorf("budget: empty vendor cap")
		}
		if cap < 0 {
			return nil, fmt.Errorf("budget: cap for %s is negative", vendor)
		}
		copyCaps[vendor] = cap
	}
	return &Ledger{caps: copyCaps, hardCap: total, spent: make(map[vocab.VendorName]float64), reserved: make(map[vocab.VendorName]float64)}, nil
}

// Reserve holds estimate against the vendor and total caps.
func (l *Ledger) Reserve(vendor vocab.VendorName, estimate float64) (*Reservation, error) {
	if estimate < 0 {
		return nil, fmt.Errorf("budget: negative estimate")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	remaining := l.remainingLocked(vendor)
	totalRemaining := l.totalRemainingLocked()
	vendorCapped := l.caps[vendor] > 0
	if (vendorCapped && estimate > remaining) || (l.hardCap > 0 && estimate > totalRemaining) {
		return nil, &CapError{Vendor: vendor, Requested: estimate, Remaining: minPositive(remaining, totalRemaining, l.hardCap > 0)}
	}
	l.reserved[vendor] += estimate
	return &Reservation{ledger: l, vendor: vendor, estimate: estimate}, nil
}

// Costs returns a copy suitable for status and debug reporting.
func (l *Ledger) Costs() Snapshot {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.snapshotLocked()
}

func (l *Ledger) remainingLocked(vendor vocab.VendorName) float64 {
	cap, ok := l.caps[vendor]
	if !ok || cap == 0 {
		return 0
	}
	return cap - l.spent[vendor] - l.reserved[vendor]
}

func (l *Ledger) totalRemainingLocked() float64 {
	return l.hardCap - sum(l.spent) - sum(l.reserved)
}

func (l *Ledger) snapshotLocked() Snapshot {
	byVendor := make(map[vocab.VendorName]VendorCost, len(l.spent)+len(l.reserved))
	for vendor, value := range l.spent {
		byVendor[vendor] = VendorCost{Spent: value, Reserved: l.reserved[vendor]}
	}
	for vendor, value := range l.reserved {
		item := byVendor[vendor]
		item.Reserved = value
		byVendor[vendor] = item
	}
	return Snapshot{Spent: sum(l.spent), Reserved: sum(l.reserved), ByVendor: byVendor}
}

func sum(values map[vocab.VendorName]float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func minPositive(vendor, total float64, bounded bool) float64 {
	if !bounded || vendor <= 0 {
		return total
	}
	if total <= 0 || vendor < total {
		return vendor
	}
	return total
}

// Reservation represents one in-flight vendor call.
type Reservation struct {
	ledger   *Ledger
	vendor   vocab.VendorName
	estimate float64
	done     bool
}

// Settle releases the estimate and records actual spend. It is idempotent;
// subsequent calls return the original settled amount.
func (r *Reservation) Settle(actual float64) (float64, error) {
	if actual < 0 {
		return 0, fmt.Errorf("budget: negative actual cost")
	}
	r.ledger.mu.Lock()
	defer r.ledger.mu.Unlock()
	if r.done {
		return actual, nil
	}
	if actual > r.estimate {
		extra := actual - r.estimate
		if cap := r.ledger.caps[r.vendor]; cap > 0 && r.ledger.spent[r.vendor]+actual > cap {
			return 0, &CapError{Vendor: r.vendor, Requested: extra, Remaining: cap - r.ledger.spent[r.vendor]}
		}
		if r.ledger.hardCap > 0 && r.ledger.totalSettledLocked()+actual > r.ledger.hardCap {
			return 0, &CapError{Vendor: r.vendor, Requested: extra, Remaining: r.ledger.hardCap - r.ledger.totalSettledLocked()}
		}
	}
	r.ledger.reserved[r.vendor] -= r.estimate
	r.ledger.spent[r.vendor] += actual
	r.done = true
	return actual, nil
}

// Release cancels an unused reservation without adding spend.
func (r *Reservation) Release() {
	r.ledger.mu.Lock()
	defer r.ledger.mu.Unlock()
	if r.done {
		return
	}
	r.ledger.reserved[r.vendor] -= r.estimate
	r.done = true
}

func (l *Ledger) totalSettledLocked() float64 { return sum(l.spent) }
