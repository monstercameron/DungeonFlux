package domain

// KillCamView is a bounded replay of an authoritative combat defeat. An empty
// URL means no cinematic is active; the battlefield remains the fallback.
type KillCamView struct {
	URL        string
	Sequence   uint64
	Outcome    string
	Attacker   string
	Victim     string
	OffsetMS   int64
	DurationMS int64
	Playing    bool
}
