package domain

type MusicTrack struct {
	ID          string  `json:"track_id"`
	Asset       AssetID `json:"asset"`
	URL         string  `json:"url"`
	LoopStartMS int     `json:"loop_start_ms"`
	LoopEndMS   int     `json:"loop_end_ms"`
	BPM         int     `json:"bpm"`
	Level       float64 `json:"level"`
	Duck        float64 `json:"duck"`
}
type MusicCatalogue struct {
	Tracks []MusicTrack `json:"tracks"`
}
