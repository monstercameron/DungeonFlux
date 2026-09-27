package content

import (
	"fmt"
	"hash/crc32"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// MusicTrack describes one authored demo track and its sidecar metadata.
type MusicTrack struct {
	ID            string
	Asset         domain.AssetID
	Kind          string
	LengthMS      int
	BPM           int
	BarMS         int
	DownbeatMS    int
	LoopStartMS   int
	LoopEndMS     int
	Key           string
	Level         float64
	Duck          float64
	Fallback      string
	Cue           string
	Transition    string
	CrossfadeBars int
	Seed          uint32
	Seeded        bool
	Conditioned   bool
}

// MusicCatalogue is the complete authored soundtrack for the demo.
type MusicCatalogue struct {
	LibraryVersion string
	Tracks         []MusicTrack
}

// Track returns the authored track with id, if the catalogue contains it.
func (c MusicCatalogue) Track(id string) (MusicTrack, bool) {
	for _, track := range c.Tracks {
		if track.ID == id {
			return track, true
		}
	}
	return MusicTrack{}, false
}

// LobbyTracks returns the accepted lobby bed and title stinger assets.
// OPENING_SWELL is the build-time title stinger; the manifest has no separate
// title-stinger asset.
func (c MusicCatalogue) LobbyTracks() (MusicTrack, MusicTrack, bool) {
	bed, bedOK := c.Track("THEME_MAIN")
	stinger, stingerOK := c.Track("OPENING_SWELL")
	return bed, stinger, bedOK && stingerOK
}

// DefaultMusicCatalogue returns the twelve §0.19 demo tracks.
func DefaultMusicCatalogue() MusicCatalogue {
	version := "demo-v1"
	tracks := []MusicTrack{
		{ID: "THEME_MAIN", Asset: "THEME_MAIN", Kind: "loop", LengthMS: 96000, BPM: 80, BarMS: 3000, LoopEndMS: 96000, Key: "D dorian", Level: 0.6, Duck: 0.6, Fallback: "royalty_free_fantasy_track", Cue: "lobby", Transition: "bar", CrossfadeBars: 1, Seeded: true},
		{ID: "CREATION_BED_LOOP", Asset: "music_creation_bed_loop", Kind: "loop", LengthMS: 96000, BPM: 80, BarMS: 3000, LoopEndMS: 96000, Key: "D dorian", Level: 0.35, Duck: 0.3, Fallback: "THEME_MAIN", Cue: "creation", Transition: "bar", CrossfadeBars: 1, Seeded: true, Conditioned: true},
		{ID: "OPENING_SWELL", Asset: "OPENING_SWELL", Kind: "one_shot", LengthMS: 12000, BPM: 80, BarMS: 3000, Key: "D dorian", Level: 1, Duck: 0.3, Fallback: "TAVERN_WARM_LOOP", Cue: "opening", Transition: "crossfade_at_6000ms", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "TAVERN_WARM_LOOP", Asset: "music_tavern_warm_loop", Kind: "loop", LengthMS: 96000, BPM: 80, BarMS: 3000, LoopEndMS: 96000, Key: "D dorian / F major", Level: 0.3, Duck: 0.3, Fallback: "THEME_MAIN", Cue: "tavern", Transition: "bar", CrossfadeBars: 1, Seeded: true, Conditioned: true},
		{ID: "STING_STRANGER", Asset: "music_sting_stranger", Kind: "stinger", LengthMS: 6000, BPM: 0, Key: "D phrygian", Level: 1, Duck: 0.15, Fallback: "sfx_stranger_sting", Cue: "stranger", Transition: "after_line", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "STING_COMBAT_START", Asset: "music_sting_combat_start", Kind: "stinger", LengthMS: 4000, BPM: 160, BarMS: 1500, Key: "D minor", Level: 1, Duck: 0.2, Fallback: "sfx_door_burst", Cue: "combat_start", Transition: "after_line", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "COMBAT_SKIRMISH_LOOP", Asset: "music_combat_skirmish_loop", Kind: "loop", LengthMS: 96000, BPM: 160, BarMS: 1500, LoopEndMS: 96000, Key: "D minor", Level: 0.5, Duck: 0.2, Fallback: "THEME_MAIN", Cue: "combat", Transition: "bar", CrossfadeBars: 1, Seeded: true, Conditioned: true},
		{ID: "STING_VICTORY", Asset: "music_sting_victory", Kind: "stinger", LengthMS: 6000, BPM: 0, Key: "D major", Level: 1, Duck: 0.3, Fallback: "sfx_success", Cue: "victory", Transition: "after_line", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "STING_BELL_TOLL", Asset: "music_sting_bell_toll", Kind: "stinger", LengthMS: 4000, BPM: 0, Key: "D", Level: 1, Duck: 0.3, Fallback: "sfx_cliffhanger_hit", Cue: "bell_toll", Transition: "after_line", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "CLIFF_TENSION_BED", Asset: "music_cliff_tension_bed", Kind: "one_shot", LengthMS: 24000, BPM: 60, BarMS: 4000, Key: "D phrygian", Level: 0.3, Duck: 0.3, Fallback: "silence_plus_ambience", Cue: "cliffhanger", Transition: "fade_300ms", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "STING_CLIFF_HIT", Asset: "music_sting_cliff_hit", Kind: "stinger", LengthMS: 6000, BPM: 0, Key: "D", Level: 1, Duck: 0.3, Fallback: "sfx_cliffhanger_hit", Cue: "cliff_hit", Transition: "after_line", CrossfadeBars: 0, Seeded: true, Conditioned: true},
		{ID: "END_CARD_THEME", Asset: "music_end_card_theme", Kind: "one_shot", LengthMS: 24000, BPM: 80, BarMS: 3000, Key: "D dorian", Level: 0.7, Duck: 0.7, Fallback: "THEME_MAIN", Cue: "end_card", Transition: "fade_4000ms", CrossfadeBars: 0, Seeded: false, Conditioned: true},
	}
	for i := range tracks {
		if tracks[i].Seeded {
			tracks[i].Seed = crc32.ChecksumIEEE([]byte(tracks[i].ID + version))
		}
	}
	return MusicCatalogue{LibraryVersion: version, Tracks: tracks}
}

// DomainTracks returns the fields consumed by the pure game engine.
func (c MusicCatalogue) DomainTracks() []domain.MusicTrack {
	tracks := make([]domain.MusicTrack, 0, len(c.Tracks))
	for _, track := range c.Tracks {
		tracks = append(tracks, domain.MusicTrack{ID: track.ID, Asset: track.Asset, LoopStartMS: track.LoopStartMS, LoopEndMS: track.LoopEndMS, BPM: track.BPM, Level: track.Level, Duck: track.Duck})
	}
	return tracks
}

// Validate checks IDs, timing, tempo family, fallbacks, and seed metadata.
func (c MusicCatalogue) Validate() error {
	if c.LibraryVersion == "" || len(c.Tracks) != 12 {
		return fmt.Errorf("music catalogue must contain a version and 12 tracks")
	}
	seen := make(map[string]bool, len(c.Tracks))
	for _, track := range c.Tracks {
		if track.ID == "" || track.Asset == "" || seen[track.ID] || track.Fallback == "" {
			return fmt.Errorf("invalid music track %q", track.ID)
		}
		seen[track.ID] = true
		if track.LengthMS < 3000 || track.Level < 0 || track.Level > 1 || track.Duck < 0 || track.Duck > 1 {
			return fmt.Errorf("invalid audio metadata for %q", track.ID)
		}
		if track.Kind == "loop" && (track.LengthMS != 96000 || track.LoopEndMS-track.LoopStartMS != 96000 || track.CrossfadeBars != 1) {
			return fmt.Errorf("loop %q must be a 96 second one-bar-crossfaded loop", track.ID)
		}
		if track.Kind != "stinger" && (track.BPM == 0 || track.BarMS != 240000/track.BPM) {
			return fmt.Errorf("track %q has invalid tempo metadata", track.ID)
		}
		if track.BPM != 0 && track.BPM != 60 && track.BPM != 80 && track.BPM != 120 && track.BPM != 160 {
			return fmt.Errorf("track %q uses an unsupported tempo", track.ID)
		}
		if track.Seed != 0 && track.Seed != crc32.ChecksumIEEE([]byte(track.ID+c.LibraryVersion)) {
			return fmt.Errorf("track %q has an invalid seed", track.ID)
		}
		if track.Seeded && track.Seed == 0 {
			return fmt.Errorf("seeded track %q has no seed", track.ID)
		}
	}
	return nil
}
