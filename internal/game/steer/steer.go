package steer

// Intent describes how a player's current choice relates to the destination.
type Intent uint8

const (
	// IntentOnFunnel means the players are pursuing a route to the destination.
	IntentOnFunnel Intent = iota
	// IntentCurious means the players are exploring an off-funnel choice
	// productively; it should be rewarded rather than interrupted.
	IntentCurious
	// IntentDrifting means the players have left the route without a clear goal.
	IntentDrifting
	// IntentStalled means no meaningful progress has been made on the current
	// route.
	IntentStalled
)

// Rung is one level of the funnel's softest-first steering ladder.
type Rung uint8

const (
	// RungNone means that no steering action is due.
	RungNone Rung = iota
	// RungLure offers an appealing reason to follow the destination.
	RungLure
	// RungPersonalHook brings in a problem tied to a character's backstory.
	RungPersonalHook
	// RungClueRelocation moves an unfound clue to the players' current node.
	RungClueRelocation
	// RungConsequence shows the cost of an ignored threat.
	RungConsequence
	// RungWorldClosesIn brings the antagonist to the players as a last resort.
	RungWorldClosesIn
)

// Decision is the deterministic steering result for one beat boundary.
type Decision struct {
	Rung Rung
}

// Input is the engine-owned observation used to choose a steering rung.
type Input struct {
	// ElapsedBeats counts beats since the last meaningful progress toward the
	// destination. Negative values are treated as zero.
	ElapsedBeats int
	Intent       Intent
}

// Select chooses the softest rung licensed by the input. Each ignored beat
// licenses at most one additional rung. On-funnel and curious play never
// receives a steering intervention, regardless of elapsed beats.
func Select(input Input) Decision {
	if input.Intent == IntentOnFunnel || input.Intent == IntentCurious {
		return Decision{Rung: RungNone}
	}
	beats := input.ElapsedBeats
	if beats < 0 {
		beats = 0
	}
	if input.Intent == IntentStalled && beats == 0 {
		beats = 1
	}
	if beats > int(RungWorldClosesIn) {
		beats = int(RungWorldClosesIn)
	}
	return Decision{Rung: Rung(beats)}
}

// String returns the stable event-log name for a rung.
func (r Rung) String() string {
	switch r {
	case RungLure:
		return "lure"
	case RungPersonalHook:
		return "personal_hook"
	case RungClueRelocation:
		return "clue_relocation"
	case RungConsequence:
		return "consequence"
	case RungWorldClosesIn:
		return "world_closes_in"
	default:
		return "none"
	}
}
