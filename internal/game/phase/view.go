package phase

import (
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/check"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// View returns the detached projection owned by the active phase.
func (m Machine) View() domain.View {
	spotlight := m.spotlight
	if m.State() == vocab.StateCombat && m.combat.TurnSeat >= 1 && m.combat.TurnSeat <= 2 {
		spotlight = domain.SeatID(m.combat.TurnSeat)
	}
	view := domain.View{Path: m.State(), Paused: m.paused, Spotlight: spotlight, Seats: m.viewSeats()}
	m.decorateKillcam(&view)
	if m.State() == vocab.StateOpening {
		openingView := m.opening.View()
		openingView.Path, openingView.Paused = m.State(), m.paused
		openingView.Spotlight, openingView.Seats = spotlight, m.viewSeats()
		return openingView
	}
	if m.State() == vocab.StateCombat {
		view.Combat = m.combatView()
		view.Battlefield = m.combatBattlefieldView(*view.Combat)
		m.decorateCombatMap(&view)
	}
	if m.State() == vocab.StateCheck || m.State() == vocab.StateResolution {
		view.Dice = m.checkDiceView()
	}
	return view
}

// checkDiceView projects the active persuasion check onto the shared DiceView
// so the DM and both phones (INT-009) can show the rolling animation and,
// once check.Resolved, the kept face, total, and outcome. The state stays
// StateCheck for the whole roll, so this always reports "rolling" there; the
// outcome only appears once the phase machine has moved on to StateResolution
// and check.Machine.State() reports Resolved.
func (m Machine) checkDiceView() *domain.DiceView {
	view := &domain.DiceView{Kind: "check", Modifier: m.check.Modifier(), DC: m.check.DC()}
	switch m.check.State() {
	case check.Rolling:
		view.State = "rolling"
		return view
	case check.Resolved:
		view.State = "resolved"
	default:
		view.State = "offered"
		return view
	}
	outcome, ok := m.check.Outcome()
	if !ok {
		return view
	}
	view.D20, view.Modifier, view.DC = outcome.Roll.Kept, outcome.Modifier, outcome.DC
	if outcome.Success {
		view.Outcome = "success"
	} else {
		view.Outcome = "failure"
	}
	return view
}

// LegalMoveViews returns the action menu for one seat in the active phase.
func (m Machine) LegalMoveViews(seat domain.SeatID) []domain.MoveView {
	if seat < 1 || seat > 2 || m.paused {
		return nil
	}
	card, _ := seatFrom(m.seats, seat)
	switch m.State() {
	case vocab.StateLobby:
		return []domain.MoveView{enabledMove(vocab.MoveReady, true, "")}
	case vocab.StateCreation:
		state, ok := m.creation.Seat(seat)
		if !ok {
			return nil
		}
		return creation.LegalMoveViews(state)
	case vocab.StateExploration:
		if m.conversationDone {
			return []domain.MoveView{
				move(vocab.MoveLeave, "Leave the tavern", true, ""),
				move(vocab.MoveTalkVell, "Talk to Mother Vell", false, "The conversation is over"),
			}
		}
		active := m.spotlight == seat
		return []domain.MoveView{move(vocab.MoveTalkVell, "Talk to Mother Vell", active, waitingReason(m.spotlight)), move(vocab.MoveLeave, "Leave", active, waitingReason(m.spotlight))}
	case vocab.StateConversation:
		active := m.spotlight == seat
		return []domain.MoveView{move(vocab.MovePersuade, "Persuade +4 vs DC 10", active, waitingReason(m.spotlight)), move(vocab.MoveStepAway, "Step away", active, waitingReason(m.spotlight))}
	case vocab.StateCheck:
		active := m.spotlight == seat && m.check.State() == check.Offered
		return []domain.MoveView{move(vocab.MovePersuade, "Persuade +4 vs DC 10", active, waitingReason(m.spotlight))}
	case vocab.StateCombat:
		return m.combatMoves(seat, card)
	default:
		return nil
	}
}

func (m Machine) combatMoves(seat domain.SeatID, card domain.SeatView) []domain.MoveView {
	if m.killcam.URL != "" {
		return nil
	}
	active := m.combat.TurnSeat == int(seat) && m.combat.Phase == "pc_turn"
	alive := m.combat.Thrall.HP > 0
	waiting := waitingReason(domain.SeatID(m.combat.TurnSeat))
	moveView := move(vocab.MoveMove, "Move", active, waiting)
	if active {
		moveView.Options = reachableOptions(m.combat, seat)
		if len(moveView.Options) == 0 {
			moveView.Enabled = false
			moveView.Reason = "No movement left"
		}
	}
	attack := move(vocab.MoveAttack, "Attack the drowned thrall", active && card.Character != nil && alive, waiting)
	attack.TargetID = domain.EntityID(m.combat.Thrall.ID)
	if active && card.Character == nil {
		attack.Reason = "Building your hero…"
	} else if active && !alive {
		attack.Reason = "The drowned thrall is defeated"
	}
	end := move(vocab.MoveEndTurn, "End turn", active && alive, waiting)
	return m.withCombatMoveUI(seat, []domain.MoveView{moveView, attack, end})
}

func (m Machine) combatView() *domain.CombatView {
	presentation := m.combat.Presentation
	view := &domain.CombatView{
		Round:   m.combat.TurnNumber,
		Banner:  string(m.combat.Phase),
		Camera:  domain.CameraView{Preset: presentation.Camera.Preset, FocusTokenID: domain.TokenID(presentation.Camera.FocusTokenID), Seq: presentation.Camera.Seq, Follow: presentation.Camera.Follow, DurationMS: presentation.Camera.DurationMS},
		Contact: domain.TimerView{Name: "contact", RemainingMS: presentation.ContactMS, TotalMS: presentation.ContactTotalMS},
		Shake:   domain.ShakeView{AmplitudePX: presentation.Shake.AmplitudePX, DurationMS: presentation.Shake.DurationMS, Seq: presentation.Shake.Seq},
	}
	for _, pc := range m.combat.PCs {
		visual := presentation.Tokens[pc.ID]
		name := pc.Name
		if name == "" {
			name = pc.ID
		}
		active := m.combat.Phase == combat.PCTurn && pc.Seat == m.combat.TurnSeat
		view.Tokens = append(view.Tokens, domain.TokenView{ID: domain.TokenID(pc.ID), Kind: combatKind(string(pc.Build.Class), pc.Species), Name: name, Portrait: domain.AssetID(pc.Portrait), Cell: domain.Cell{R: pc.Position.Y, C: pc.Position.X}, Path: domainCells(visual.Path), Anim: visual.Anim, AnimSeq: visual.AnimSeq, StepMS: visual.StepMS, HP: pc.HP, HPMax: pc.MaxHP, Active: active})
		view.TurnOrder = append(view.TurnOrder, domain.TurnEntry{TokenID: domain.TokenID(pc.ID), Name: name, Portrait: domain.AssetID(pc.Portrait), HP: pc.HP, HPMax: pc.MaxHP, Active: active, Done: pc.IsDown()})
	}
	thrallVisual := presentation.Tokens[m.combat.Thrall.ID]
	thrallActive := m.combat.Phase == combat.EnemyTurn
	view.Tokens = append(view.Tokens, domain.TokenView{ID: domain.TokenID(m.combat.Thrall.ID), Kind: "thrall", Name: m.combat.Thrall.ID, Cell: domain.Cell{C: m.combat.ThrallPosition.X, R: m.combat.ThrallPosition.Y}, Path: domainCells(thrallVisual.Path), Anim: thrallVisual.Anim, AnimSeq: thrallVisual.AnimSeq, StepMS: thrallVisual.StepMS, HP: m.combat.Thrall.HP, HPMax: m.combat.Thrall.MaxHP, Active: thrallActive})
	view.TurnOrder = append(view.TurnOrder, domain.TurnEntry{TokenID: domain.TokenID(m.combat.Thrall.ID), Name: m.combat.Thrall.ID, HP: m.combat.Thrall.HP, HPMax: m.combat.Thrall.MaxHP, Active: thrallActive, Done: m.combat.Thrall.HP <= 0})
	for _, highlight := range presentation.Highlights {
		view.Highlights = append(view.Highlights, domain.HighlightView{Kind: highlight.Kind, Cells: domainCells(highlight.Cells)})
	}
	return view
}

func combatKind(class, species string) string {
	class = strings.ToLower(strings.TrimSpace(class))
	if class == "" {
		return "pc"
	}
	kind := "pc-" + class
	if species = strings.ToLower(strings.TrimSpace(species)); species != "" {
		kind += "-" + species
	}
	return kind
}

func (m Machine) combatBattlefieldView(combatView domain.CombatView) *domain.BattlefieldView {
	source := m.oneShot.Encounter.Battlefield
	return &domain.BattlefieldView{
		Mode: source.Mode, Visible: true, SceneURL: source.SceneURL, LiteURL: source.LiteURL,
		Transform: source.Transform, Cameras: cloneBattlefieldCameras(source.Cameras), Grid: cloneGrid(source.Grid), Flat: source.Flat,
		Camera: combatView.Camera, Tokens: append([]domain.TokenView(nil), combatView.Tokens...),
		Highlights: append([]domain.HighlightView(nil), combatView.Highlights...), TurnOrder: append([]domain.TurnEntry(nil), combatView.TurnOrder...),
		Round: combatView.Round, Contact: combatView.Contact, Shake: combatView.Shake,
	}
}

func domainCells(cells []combat.Cell) []domain.Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]domain.Cell, len(cells))
	for index, cell := range cells {
		out[index] = domain.Cell{C: cell.X, R: cell.Y}
	}
	return out
}

func cloneGrid(source domain.Grid) domain.Grid {
	source.Walkable = append([]bool(nil), source.Walkable...)
	return source
}

func cloneBattlefieldCameras(source map[string]domain.CameraDef) map[string]domain.CameraDef {
	if source == nil {
		return nil
	}
	out := make(map[string]domain.CameraDef, len(source))
	maps.Copy(out, source)
	return out
}

func creationSeatView(state creation.SeatState) domain.SeatView {
	seat := domain.SeatView{Seat: state.Seat, PlayerNumber: int(state.Seat)}
	if !state.Built {
		return seat
	}
	name := state.Flavor.Name
	if name == "" {
		name = "Hero " + strconv.Itoa(int(state.Seat))
	}
	portrait := domain.AssetID("")
	if state.Species != "" {
		portrait = domain.AssetID("ui/species_" + strings.ToLower(state.Species))
	}
	seat.Build = &domain.BuildCard{Name: name, Class: string(state.Class), Portrait: portrait, PlayerNumber: int(state.Seat), Stats: buildStats(state.Build)}
	seat.Character = &domain.Character{ID: domain.EntityID("pc-" + strconv.Itoa(int(state.Seat))), Name: name, Class: string(state.Class), Species: state.Species, Gender: state.Gender, Portrait: portrait, Hook: state.Flavor.Hook, PersuasionModifier: state.Build.PersuasionBonus, HP: state.Build.HP, MaxHP: state.Build.MaxHP, AC: state.Build.AC}
	return seat
}

func buildStats(build rules.Build) *domain.BuildStats {
	return &domain.BuildStats{
		Abilities:          [6]int{build.Abilities.Strength, build.Abilities.Dexterity, build.Abilities.Constitution, build.Abilities.Intelligence, build.Abilities.Wisdom, build.Abilities.Charisma},
		SaveProficiencies:  append([]string(nil), build.SaveProficiencies...),
		SkillProficiencies: cloneStringMap(build.SkillProficiencies),
		HP:                 build.HP, MaxHP: build.MaxHP, AC: build.AC,
		AttackName: build.Attack.Name, AttackDice: build.Attack.Dice, AttackDamageType: build.Attack.DamageType, AttackBonus: build.AttackBonus,
		Equipment: equipmentItems(build.Class),
	}
}

func equipmentItems(class rules.Class) []domain.EquipmentItem {
	items := rules.Equipment(class)
	views := make([]domain.EquipmentItem, 0, len(items))
	for _, item := range items {
		views = append(views, domain.EquipmentItem{Name: item.Name, Description: item.Description, Slot: item.Slot, Worn: item.Worn})
	}
	return views
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	maps.Copy(clone, values)
	return clone
}

func (m Machine) viewSeats() []domain.SeatView {
	seats := cloneSeats(m.seats)
	for index := range seats {
		seats[index].Moves = m.LegalMoveViews(seats[index].Seat)
	}
	return seats
}

func enabledMove(id vocab.MoveID, enabled bool, disabledReason string) domain.MoveView {
	return move(id, moveLabel(id), enabled, disabledReason)
}

func move(id vocab.MoveID, label string, enabled bool, disabledReason string) domain.MoveView {
	if enabled {
		disabledReason = ""
	}
	return domain.MoveView{ID: id, Label: label, Enabled: enabled, Reason: disabledReason}
}

func waitingReason(seat domain.SeatID) string {
	if seat < 1 || seat > 2 {
		return "Waiting for your turn"
	}
	return fmt.Sprintf("Waiting for seat %d", seat)
}

func moveLabel(id vocab.MoveID) string {
	switch id {
	case vocab.MoveReady:
		return "Ready"
	case vocab.MoveSpecies:
		return "Choose a species"
	case vocab.MoveGender:
		return "Choose a gender"
	case vocab.MoveRollHero:
		return "Roll my hero"
	case vocab.MoveTalkVell:
		return "Talk to Mother Vell"
	case vocab.MovePersuade:
		return "Persuade +4 vs DC 10"
	case vocab.MoveStepAway:
		return "Step away"
	case vocab.MoveLeave:
		return "Leave"
	case vocab.MoveAttack:
		return "Attack the drowned thrall"
	case vocab.MoveMove:
		return "Move"
	case vocab.MoveEndTurn:
		return "End turn"
	default:
		return string(id)
	}
}

func reachableOptions(state combat.State, seat domain.SeatID) []domain.OptionView {
	options := make([]domain.OptionView, 0)
	for _, entry := range state.ReachFor(int(seat)).Cells {
		if entry.Dash {
			continue
		}
		column, row := entry.Cell.X, entry.Cell.Y
		options = append(options, domain.OptionView{ID: fmt.Sprintf("%d,%d", column, row), Label: fmt.Sprintf("(%d, %d)", column, row)})
	}
	return options
}

func seatFrom(seats []domain.SeatView, wanted domain.SeatID) (domain.SeatView, bool) {
	for _, seat := range seats {
		if seat.Seat == wanted {
			return seat, true
		}
	}
	return domain.SeatView{}, false
}

func cloneSeats(seats []domain.SeatView) []domain.SeatView {
	out := append([]domain.SeatView(nil), seats...)
	for index := range out {
		out[index].Moves = append([]domain.MoveView(nil), seats[index].Moves...)
		for moveIndex := range out[index].Moves {
			out[index].Moves[moveIndex].Options = append([]domain.OptionView(nil), seats[index].Moves[moveIndex].Options...)
		}
	}
	return out
}
