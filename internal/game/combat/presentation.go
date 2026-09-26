package combat

// initPresentation establishes the first deterministic render snapshot.
func (s *State) initPresentation() {
	s.Presentation.Tokens = make(map[string]TokenPresentation, 3)
	for _, id := range []string{s.PCs[0].ID, s.PCs[1].ID, s.Thrall.ID} {
		s.Presentation.Tokens[id] = TokenPresentation{Anim: "idle", AnimSeq: 1}
	}
	s.Presentation.Camera = CameraPresentation{Preset: "COMBAT_EST", Seq: 1}
}

func (s *State) setCamera(preset, focus string, follow bool, durationMS int64) {
	s.Presentation.Camera = CameraPresentation{
		Preset: preset, FocusTokenID: focus, Follow: follow, DurationMS: durationMS,
		Seq: s.Presentation.Camera.Seq + 1,
	}
}

func (s *State) setAction(token, anim string, path []Cell) {
	if s.Presentation.Tokens == nil {
		s.Presentation.Tokens = make(map[string]TokenPresentation, 3)
	}
	visual := s.Presentation.Tokens[token]
	visual.Path = append([]Cell(nil), path...)
	visual.Anim = anim
	visual.AnimSeq++
	s.Presentation.Tokens[token] = visual
}

func (s *State) clearPaths() {
	for id, visual := range s.Presentation.Tokens {
		visual.Path = nil
		s.Presentation.Tokens[id] = visual
	}
}

func (s *State) setContact(contactMS, totalMS int64) {
	if contactMS < 0 {
		contactMS = 0
	}
	if totalMS < contactMS {
		totalMS = contactMS
	}
	s.Presentation.ContactMS = contactMS
	s.Presentation.ContactTotalMS = totalMS
}

func (s *State) clearContact() {
	s.Presentation.ContactMS = 0
	s.Presentation.ContactTotalMS = 0
}

func (s *State) setShake(amplitude float64, durationMS int64) {
	s.Presentation.Shake = ShakePresentation{
		AmplitudePX: amplitude, DurationMS: durationMS,
		Seq: s.Presentation.Shake.Seq + 1,
	}
}

func (s *State) setReachHighlights(participant Participant) {
	cells := make([]Cell, 0)
	for row := 0; row < s.Grid.Rows; row++ {
		for column := 0; column < s.Grid.Cols; column++ {
			cell := Cell{X: column, Y: row}
			if cell == participant.Position || !s.Grid.IsWalkable(cell) {
				continue
			}
			if _, ok := shortestPath(s.Grid, participant.Position, cell, maxCombatMove); ok {
				cells = append(cells, cell)
			}
		}
	}
	s.Presentation.Highlights = []Highlight{{Kind: "reach", Cells: cells}}
}

func (s *State) clearHighlights() { s.Presentation.Highlights = nil }

func (s *State) finishPresentation(slain bool) {
	s.clearContact()
	s.clearHighlights()
	s.clearPaths()
	if slain {
		s.setAction(s.Thrall.ID, "fall", nil)
		s.setCamera("VICTORY", s.Thrall.ID, false, 800)
		return
	}
	s.setAction(s.Thrall.ID, "flee", nil)
	s.setCamera("COMBAT_EST", s.Thrall.ID, false, 800)
}
