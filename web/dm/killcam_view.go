package dm

// killcamNeedsSeek keeps the persistent video aligned with fresh snapshots.
// Repeated snapshots must not rewind a clip that continued playing locally.
func killcamNeedsSeek(previous, next KillcamModel, currentMS float64) bool {
	if next.Key == "" {
		return false
	}
	if previous.Key != next.Key || previous.Playing != next.Playing {
		return true
	}
	if previous.OffsetMS == next.OffsetMS {
		return false
	}
	drift := currentMS - float64(next.OffsetMS)
	return drift < -250 || drift > 250
}
