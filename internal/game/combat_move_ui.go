package game

// WithCombatMoveUI configures the combat_move_ui flag: with it on, the
// active seat can move or dash on the phone's top-down combat map.
func WithCombatMoveUI(enabled bool) Option {
	return func(state *State) {
		state.phase.ConfigureCombatMoveUI(enabled)
	}
}
