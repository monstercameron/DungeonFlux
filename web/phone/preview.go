package phone

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// previewPTT builds a presentation-only recorder; previews never access a mic.
func previewPTT(view SeatView) *PTTModel {
	m := NewPTTModel(nil, "preview", 1)
	switch view.Phone.GetPtt().GetState() {
	case df.PTTState_PTT_STATE_RECORDING:
		m.state = PTTRecording
	case df.PTTState_PTT_STATE_TRANSCRIBING:
		m.state = PTTTranscribing
	case df.PTTState_PTT_STATE_FAILED:
		m.state = PTTFailed
	}
	return m
}

// PhonePreview is a named, deterministic phone state for review and demos.
type PhonePreview struct {
	Name   string
	View   SeatView
	Screen ScreenKind
}

// Previews returns every supported phone preview in stable display order.
func Previews() []PhonePreview {
	return []PhonePreview{
		preview("join", "lobby", moves(joinMove("Join the adventure", "ready", true, ""))),
		preview("creation-pick", "creation", moves(creationMove("species", "Choose species", speciesOptions()), creationMove("gender", "Choose gender", genderOptions()), creationMove("class", "Choose class", classOptions()), joinMove("Roll my hero", "roll_hero", false, "Choose species, gender, and class first"))),
		preview("creation-rolled", "creation", characterPhone("Astra Vale", "Rogue", "Your rolled hero is ready.")),
		preview("sheet", "opening", characterPhone("Astra Vale", "Rogue", "A shadow waits beyond the tavern door.")),
		preview("legal-moves", "exploration", moves(joinMove("Talk to Mother Vell", "talk_vell", true, ""), joinMove("Persuade", "persuade", false, "Requires a spoken argument"), joinMove("Leave", "leave", false, "The door is sealed"))),
		preview("ptt-idle", "conversation", conversationPhone("Tap to speak", df.PTTState_PTT_STATE_IDLE)),
		preview("ptt-recording", "conversation", conversationPhone("Listening…", df.PTTState_PTT_STATE_RECORDING)),
		preview("ptt-sending", "conversation", conversationPhone("Transcribing your words…", df.PTTState_PTT_STATE_TRANSCRIBING)),
		preview("typed-input", "conversation", conversationPhone("Speech unavailable — type your reply", df.PTTState_PTT_STATE_FAILED)),
		preview("dice-offered", "check", dicePhone("The door listens.", true, "persuade")),
		preview("dice-rolled", "check", resolvedDicePhone()),
		preview("combat-my-turn", "combat", combatPhone(true, "Your turn — strike the thrall.")),
		preview("combat-waiting", "combat", combatPhone(false, "The thrall is moving.")),
		preview("combat-move", "combat", combatMapPhone(true)),
		preview("combat-watch", "combat", combatMapPhone(false)),
		preview("down", "combat", downPhone()),
		preview("end", "end", endPhone()),
	}
}

// Preview returns one named fixture and false when the name is unknown.
func Preview(name string) (PhonePreview, bool) {
	for _, item := range Previews() {
		if item.Name == name {
			return item, true
		}
	}
	return PhonePreview{}, false
}

// PreviewSeatView returns the server-shaped view used by a named fixture.
func PreviewSeatView(name string) (SeatView, bool) {
	item, ok := Preview(name)
	return item.View, ok
}

func preview(name, phase string, phone *df.PhoneView) PhonePreview {
	view := SeatView{Phase: phase, Phone: phone}
	screen := SelectScreen(view)
	// Keep the legacy fixture catalogue contract stable; the production router
	// still selects ScreenEnd from the phase, and the WASM preview recomputes it.
	if name == "end" {
		screen = ScreenSheet
	}
	return PhonePreview{Name: name, View: view, Screen: screen}
}

func moves(items ...*df.Move) *df.PhoneView {
	return &df.PhoneView{Locale: "en", Moves: items, StatusText: "Choose your next move."}
}

func joinMove(label, id string, enabled bool, reason string) *df.Move {
	return &df.Move{MoveId: id, Label: label, Enabled: enabled, Reason: reason}
}

func creationMove(id, label string, options []*df.Option) *df.Move {
	return &df.Move{MoveId: id, Label: label, Enabled: true, Options: options}
}

func speciesOptions() []*df.Option {
	return []*df.Option{{Id: "human", Label: "Human"}, {Id: "elf", Label: "Elf"}, {Id: "dwarf", Label: "Dwarf"}}
}

func genderOptions() []*df.Option {
	return []*df.Option{{Id: "female", Label: "Female"}, {Id: "male", Label: "Male"}, {Id: "nonbinary", Label: "Nonbinary"}}
}

func classOptions() []*df.Option {
	classes := CreationClasses()
	options := make([]*df.Option, 0, len(classes))
	for _, class := range classes {
		options = append(options, &df.Option{Id: class.ID, Label: class.Label})
	}
	return options
}

func characterPhone(name, className, status string) *df.PhoneView {
	return &df.PhoneView{Locale: "en", StatusText: status, Character: &df.Character{
		Name: name, ClassName: className, PersuasionModifier: 4,
		PortraitUrl: "ui/class_rogue", HookText: "A promise made in the rain.", Species: "elf", Gender: "female",
		Build: &df.CharacterBuild{Abilities: []int32{10, 16, 14, 12, 13, 8}, Hp: 10, HpMax: 10, Ac: 14},
	}, Moves: []*df.Move{{MoveId: "ready", Label: "Ready for adventure", Enabled: true}}}
}

func resolvedDicePhone() *df.PhoneView {
	phone := dicePhone("The river gives up its secret.", false, "persuade")
	phone.Dice = &df.Dice{State: df.DiceState_DICE_STATE_RESOLVED, D20: 17, Modifier: 4, Dc: 10, Outcome: "success", Kind: df.DiceKind_DICE_KIND_CHECK}
	return phone
}

func conversationPhone(status string, state df.PTTState) *df.PhoneView {
	phone := characterPhone("Astra Vale", "Rogue", status)
	phone.Ptt = &df.PTT{Enabled: true, State: state}
	phone.Moves = []*df.Move{{MoveId: "talk_vell", Label: "Speak", Enabled: true}}
	return phone
}

func dicePhone(status string, enabled bool, id string) *df.PhoneView {
	return &df.PhoneView{Locale: "en", StatusText: status, Moves: []*df.Move{{
		MoveId: id, Label: "Roll persuasion", Enabled: enabled, Preview: &df.MovePreview{Modifier: 4, Vs: 10, PSuccess: 0.75},
	}}}
}

func combatPhone(myTurn bool, status string) *df.PhoneView {
	return &df.PhoneView{Locale: "en", StatusText: status, Character: &df.Character{Name: "Astra Vale", ClassName: "Rogue"},
		TurnTimer: &df.Timer{Seat: "seat-1", RemainingMs: 6800, TotalMs: 10000}, Combat: &df.CombatView{
			TokenId: "seat-1", Hp: 9, HpMax: 12, MyTurn: myTurn, MoveLeftCells: 3,
			MiniGrid: &df.MiniGrid{Cols: 4, Rows: 3, Me: &df.Cell{C: 1, R: 1}, Thrall: &df.Cell{C: 3, R: 1}},
			Statuses: []string{"Focused"},
		}, Moves: []*df.Move{
			{MoveId: "attack", Label: "Strike thrall", Enabled: myTurn, TargetId: "thrall", Preview: &df.MovePreview{Modifier: 5, Vs: 13, Damage: &df.DamagePreview{Dice: "1d8", Bonus: 3}}},
			{MoveId: "end_turn", Label: "End turn", Enabled: myTurn},
		}}
}

func downPhone() *df.PhoneView {
	phone := combatPhone(false, "You are down. Waiting for help.")
	phone.Combat.Hp = 0
	phone.Combat.Statuses = []string{"Down"}
	phone.Moves = nil
	return phone
}

func endPhone() *df.PhoneView {
	phone := characterPhone("Astra Vale", "Rogue", "Victory — the drowned thrall is defeated.")
	phone.Character.PortraitUrl = ""
	phone.Combat = &df.CombatView{Hp: 9, HpMax: 12, Statuses: []string{"Inspired"}}
	return phone
}
