package dice

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

var errInvalidSides = errors.New("dice sides must be positive")

// Roll is one raw die result and the counter at which it was produced.
type Roll struct {
	Counter uint64
	Sides   int
	Face    int
}

// Roller is a deterministic counter-based die source.
type Roller struct {
	seed    [32]byte
	counter uint64
	forced  *int
}

// New returns a deterministic roller for seed. Seeds shorter than 32 bytes are
// hashed; longer seeds are truncated to the first 32 bytes.
func New(seed []byte) *Roller {
	var r Roller
	if len(seed) == len(r.seed) {
		copy(r.seed[:], seed)
	} else {
		r.seed = sha256.Sum256(seed)
	}
	return &r
}

// Counter reports the index of the next draw.
func (r *Roller) Counter() uint64 { return r.counter }

// ForceD20 sets the next d20 face. It replaces an earlier force and is
// consumed by the next RollD20 call, including an advantage roll.
func (r *Roller) ForceD20(face int) error {
	if face < 1 || face > 20 {
		return errors.New("forced d20 must be between 1 and 20")
	}
	r.forced = &face
	return nil
}

// Roll draws one unbiased die. Each accepted hash candidate advances the
// counter only when a face is returned; rejected candidates use the next hash
// block index without changing the externally visible draw counter.
func (r *Roller) Roll(sides int) (Roll, error) {
	if sides <= 0 {
		return Roll{}, errInvalidSides
	}
	counter := r.counter
	limit := ^uint64(0) - (^uint64(0) % uint64(sides))
	for k := uint64(0); ; k++ {
		u := r.word(counter, k)
		if u < limit {
			r.counter++
			return Roll{Counter: counter, Sides: sides, Face: int(u%uint64(sides)) + 1}, nil
		}
	}
}

// RollD20 draws a d20, optionally with advantage or disadvantage. Adv must be
// -1, 0, or +1. A forced face is the kept face and the other face is drawn
// normally, so exactly one counter is consumed by the forced result and one by
// the companion die when advantage applies.
func (r *Roller) RollD20(adv int8) (faces []int, kept Roll, source string, err error) {
	if adv < -1 || adv > 1 {
		return nil, Roll{}, "", errors.New("advantage must be -1, 0, or 1")
	}
	if r.forced != nil {
		face := *r.forced
		r.forced = nil
		forced := Roll{Counter: r.counter, Sides: 20, Face: face}
		r.counter++
		faces = []int{face}
		if adv != 0 {
			other, drawErr := r.Roll(20)
			if drawErr != nil {
				return nil, Roll{}, "", drawErr
			}
			faces = append(faces, other.Face)
		}
		return faces, forced, "FORCED", nil
	}
	first, drawErr := r.Roll(20)
	if drawErr != nil {
		return nil, Roll{}, "", drawErr
	}
	faces = []int{first.Face}
	kept = first
	if adv != 0 {
		second, secondErr := r.Roll(20)
		if secondErr != nil {
			return nil, Roll{}, "", secondErr
		}
		faces = append(faces, second.Face)
		if (adv > 0 && second.Face > first.Face) || (adv < 0 && second.Face < first.Face) {
			kept = second
		}
	}
	return faces, kept, "RNG", nil
}

func (r *Roller) word(counter, candidate uint64) uint64 {
	var input [48]byte
	copy(input[:32], r.seed[:])
	binary.LittleEndian.PutUint64(input[32:40], counter)
	binary.LittleEndian.PutUint32(input[40:44], uint32(candidate))
	sum := sha256.Sum256(input[:])
	return binary.LittleEndian.Uint64(sum[:8])
}
