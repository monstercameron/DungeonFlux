package game

import (
	"fmt"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	reasonPaused      = "Game is paused"
	reasonWaiting     = "Waiting for your turn"
	reasonNoCharacter = "Create your character first"
	reasonNoTarget    = "The drowned thrall is defeated"
	labelAttackThrall = "Attack the drowned thrall"
	labelEndTurn      = "End turn"
)

// LegalMoveViews returns the phone action menu for seat in view order. Legal
// but currently unavailable actions remain in the result with Enabled false
// and a reason suitable for display to the player.
func LegalMoveViews(view domain.View, seat domain.SeatID) []domain.MoveView {
	if seat != 1 && seat != 2 {
		return nil
	}
	if view.Paused {
		return nil
	}
	return legalMovesForPath(view, seat)
}

// AvailableActions is an alias named after the phone-facing contract in the
// product specification.
func AvailableActions(view domain.View, seat domain.SeatID) []domain.MoveView {
	return LegalMoveViews(view, seat)
}

// LegalMoveViews returns the richer phone action menu for one seat in the
// current state.
func (s *State) LegalMoveViews(seat domain.SeatID) []domain.MoveView {
	if s == nil {
		return nil
	}
	return s.phase.LegalMoveViews(seat)
}

// AvailableActions returns the current phone action menu for one seat.
func (s *State) AvailableActions(seat domain.SeatID) []domain.MoveView {
	return s.LegalMoveViews(seat)
}

func legalMovesForPath(view domain.View, seat domain.SeatID) []domain.MoveView {
	switch view.Path {
	case vocab.StateLobby:
		return []domain.MoveView{enabledMove(vocab.MoveReady, "Ready", "")}
	case vocab.StateCreation:
		return creationMoves(view, seat)
	case vocab.StateExploration:
		return explorationMoves(view, seat)
	case vocab.StateConversation:
		return conversationMoves(view, seat)
	case vocab.StateCheck:
		return checkMoves(view, seat)
	case vocab.StateCombat:
		return combatMoves(view, seat)
	default:
		return nil
	}
}

func creationMoves(view domain.View, seat domain.SeatID) []domain.MoveView {
	card, ok := seatView(view, seat)
	if !ok {
		return nil
	}
	ready := card.Character != nil
	return []domain.MoveView{
		enabledMove(vocab.MoveSpecies, "Choose species", ""),
		enabledMove(vocab.MoveGender, "Choose gender", ""),
		classMove(),
		move(vocab.MoveRollHero, "Roll my hero", card.Build == nil, reasonFor(!ready, "Roll your hero first")),
		move(vocab.MoveReady, "Ready", ready, reasonFor(!ready, "Finish your character first")),
	}
}

func classMove() domain.MoveView {
	move := enabledMove(vocab.MoveClass, "Choose class", "")
	for _, class := range rules.Classes() {
		value := string(class)
		move.Options = append(move.Options, domain.OptionView{ID: value, Label: strings.ToUpper(value[:1]) + value[1:]})
	}
	return move
}

func explorationMoves(view domain.View, seat domain.SeatID) []domain.MoveView {
	active := view.Spotlight == seat
	return []domain.MoveView{
		move(vocab.MoveTalkVell, "Talk to Mother Vell", active, reasonFor(!active, reasonWaiting)),
		move(vocab.MoveLeave, "Leave the tavern", active, reasonFor(!active, reasonWaiting)),
	}
}

func conversationMoves(view domain.View, seat domain.SeatID) []domain.MoveView {
	active := view.Spotlight == seat
	card, _ := seatView(view, seat)
	canPersuade := active && card.Character != nil
	return []domain.MoveView{
		move(vocab.MovePersuade, "Persuade", canPersuade, reasonFor(!active, reasonWaiting),
			withPreview(persuasionPreview(card.Character))),
		move(vocab.MoveStepAway, "Step away", active, reasonFor(!active, reasonWaiting)),
	}
}

func checkMoves(view domain.View, seat domain.SeatID) []domain.MoveView {
	active := view.Spotlight == seat
	label := "Roll Persuasion"
	if view.Dice != nil && view.Dice.Modifier != 0 {
		label = fmt.Sprintf("Roll Persuasion %+d vs DC %d", view.Dice.Modifier, view.Dice.DC)
	}
	return []domain.MoveView{move(vocab.MovePersuade, label, active, reasonFor(!active, reasonWaiting))}
}

func combatMoves(view domain.View, seat domain.SeatID) []domain.MoveView {
	active := view.Spotlight == seat
	if view.Combat == nil {
		return nil
	}
	thrall, alive := findThrall(view.Combat.Tokens)
	card, _ := seatView(view, seat)
	canAttack := active && card.Character != nil && alive
	attackReason := ""
	if !active {
		attackReason = reasonWaiting
	} else if card.Character == nil {
		attackReason = reasonNoCharacter
	} else if !alive {
		attackReason = reasonNoTarget
	}
	attack := move(vocab.MoveAttack, labelAttackThrall, canAttack, attackReason)
	attack.TargetID = domain.EntityID(thrall.ID)
	attack.Preview = attackPreview(card.Character, 8)
	return []domain.MoveView{
		attack,
		move(vocab.MoveMove, "Move", false, "Movement controls are unavailable"),
		move(vocab.MoveEndTurn, labelEndTurn, active && alive, reasonFor(!active, reasonWaiting)),
	}
}

func findThrall(tokens []domain.TokenView) (domain.TokenView, bool) {
	for _, token := range tokens {
		if strings.EqualFold(token.Kind, "thrall") || strings.EqualFold(string(token.ID), "thrall") {
			return token, token.HP > 0
		}
	}
	return domain.TokenView{}, false
}

func persuasionPreview(character *domain.Character) *domain.MovePreview {
	if character == nil {
		return nil
	}
	return &domain.MovePreview{Modifier: character.PersuasionModifier, VS: 10, PSuccess: successChance(character.PersuasionModifier, 10)}
}

func attackPreview(character *domain.Character, ac int) *domain.MovePreview {
	if character == nil {
		return nil
	}
	dice, bonus := weaponDamage(character.Class)
	return &domain.MovePreview{
		Modifier: attackBonus(character.Class), VS: ac,
		PSuccess: successChance(attackBonus(character.Class), ac),
		Damage:   domain.DamageView{Dice: dice, Bonus: bonus, Type: damageType(character.Class)},
	}
}

func weaponDamage(class string) (string, int) {
	switch class {
	case "Paladin":
		return "1d8", 3
	case "Rogue":
		return "1d6", 3
	case "Bard":
		return "1d4", 2
	default:
		return "1d6", 0
	}
}

func attackBonus(class string) int {
	if class == "Bard" {
		return 4
	}
	if class == "Cleric" {
		return 2
	}
	return 5
}

func damageType(class string) string {
	if class == "Paladin" {
		return "slashing"
	}
	if class == "Rogue" || class == "Bard" {
		return "piercing"
	}
	return "bludgeoning"
}

func successChance(modifier, dc int) float64 {
	needed := dc - modifier
	if needed <= 1 {
		return 1
	}
	if needed > 20 {
		return 0
	}
	return float64(21-needed) / 20
}

func seatView(view domain.View, seat domain.SeatID) (domain.SeatView, bool) {
	for _, candidate := range view.Seats {
		if candidate.Seat == seat {
			return candidate, true
		}
	}
	return domain.SeatView{}, false
}

func enabledMove(id vocab.MoveID, label, reason string) domain.MoveView {
	return move(id, label, true, reason)
}

func move(id vocab.MoveID, label string, enabled bool, reason string, options ...moveOption) domain.MoveView {
	out := domain.MoveView{ID: id, Label: label, Enabled: enabled, Reason: reason}
	for _, option := range options {
		option(&out)
	}
	return out
}

type moveOption func(*domain.MoveView)

func withPreview(preview *domain.MovePreview) moveOption {
	return func(move *domain.MoveView) { move.Preview = preview }
}

func reasonFor(condition bool, reason string) string {
	if condition {
		return reason
	}
	return ""
}
