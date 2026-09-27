package phone

import "strings"

// ExploreSnapshot is the render-safe exploration copy for the phone.
type ExploreSnapshot struct {
	Title      string
	Location   string
	Narration  string
	SceneArt   string
	Moves      []ChoiceRowModel
	StatusText string
	Locale     string
}

// NewExploreSnapshot builds the exploration scene from legal moves and status.
func NewExploreSnapshot(moves MovesSnapshot) ExploreSnapshot {
	narration := strings.TrimSpace(moves.StatusText)
	if narration == "" {
		narration = "Rain whispers against the windows of the Drowned Lantern. Something waits beyond the bar."
	}
	return ExploreSnapshot{
		Title:      "The Drowned Lantern",
		Location:   "The Drowned Lantern",
		Narration:  narration,
		SceneArt:   "tavern_interior",
		Moves:      TalkMoveRows(moves.Moves),
		StatusText: moves.StatusText,
		Locale:     moves.Locale,
	}
}

// IsExplorationMoves identifies the story moves that belong on the scene page.
func IsExplorationMoves(moves []MoveSnapshot) bool {
	for _, move := range moves {
		switch strings.ToLower(strings.TrimSpace(move.ID)) {
		case "talk_vell", "talk", "leave", "step_away", "investigate", "search":
			return true
		}
	}
	return false
}
