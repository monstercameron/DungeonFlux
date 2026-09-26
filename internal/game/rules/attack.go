package rules

import (
	"errors"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/rulings"
)

// Condition is one of the small combat statuses executed by the demo.
type Condition string

const (
	// Bloodied means a creature is at or below half its maximum HP.
	Bloodied Condition = "bloodied"
	// Down means a PC reached zero HP and is unconscious and prone.
	Down Condition = "down"
	// Prone is tracked as part of the down state in the demo.
	Prone Condition = "prone"
	// Defeated means a creature has been reduced to zero HP.
	Defeated Condition = "defeated"
	// Fled means the thrall left when the bell ended combat.
	Fled Condition = "fled"
)

// DamageRoll records one weapon or sneak-attack damage component.
type DamageRoll struct {
	Dice         string
	Faces        []int
	Bonus, Total int
	Type, Source string
}

// AttackOutcome is the complete result of a to-hit roll and its damage.
type AttackOutcome struct {
	AttackID                 string
	Attacker, Target         string
	Roll                     rulings.RollRecord
	Modifier                 int
	Breakdown                []rulings.Term
	AC                       int
	Hit, Crit                bool
	Natural                  int
	Damage                   []DamageRoll
	Total, HPBefore, HPAfter int
}

// CanSneakAttack reports whether R-D5 grants a Rogue's extra 1d6.
// The other PC must be adjacent to the target, and the target must not be Down.
func CanSneakAttack(allyAdjacent, targetDown bool) bool {
	return allyAdjacent && !targetDown
}

// CreatureState is the mutable combat data that pure rule helpers update.
type CreatureState struct {
	ID            string
	HP, MaxHP, AC int
	Conditions    []Condition
}

// Thrall returns the demo's modified Zombie: AC 8 and 12 HP.
func Thrall(id string) CreatureState { return CreatureState{ID: id, HP: 12, MaxHP: 12, AC: 8} }

// Attack resolves a d20 attack and, on a hit, rolls its damage expression.
func Attack(source *dice.Roller, attackID, attacker, target string, modifier, ac int, notation string, bonus int, damageType string, hpBefore int, adv int8) (AttackOutcome, error) {
	if source == nil {
		return AttackOutcome{}, errors.New("dice source is nil")
	}
	count, sides, err := parseDice(notation)
	if err != nil {
		return AttackOutcome{}, err
	}
	faces, kept, origin, err := source.RollD20(adv)
	if err != nil {
		return AttackOutcome{}, err
	}
	natural := 0
	if kept.Face == 1 || kept.Face == 20 {
		natural = kept.Face
	}
	roll := rulings.RollRecord{RollID: attackID + "-" + strconv.FormatUint(kept.Counter, 10), Counter: kept.Counter, Die: 20, Faces: append([]int(nil), faces...), Kept: kept.Face, Adv: adv, Source: origin}
	out := AttackOutcome{AttackID: attackID, Attacker: attacker, Target: target, Roll: roll, Modifier: modifier, Breakdown: []rulings.Term{{Label: "attack", Value: modifier}}, AC: ac, Natural: natural, HPBefore: hpBefore, HPAfter: hpBefore}
	out.Crit = kept.Face == 20
	out.Hit = kept.Face != 1 && (out.Crit || kept.Face+modifier >= ac)
	if !out.Hit {
		return out, nil
	}
	diceCount := count
	if out.Crit {
		diceCount *= 2
	}
	damage, total, err := rollDamage(source, diceCount, sides, bonus, damageType, "weapon")
	if err != nil {
		return AttackOutcome{}, err
	}
	out.Damage, out.Total = damage, total
	out.HPAfter = max(0, hpBefore-total)
	return out, nil
}

// ApplySneakAttack appends R-D5's extra 1d6 to a successful attack.
// Critical hits double the extra damage die under R-09. The outcome is updated
// in place so the combat machine can emit one complete AttackOutcome.
func ApplySneakAttack(source *dice.Roller, outcome *AttackOutcome, eligible bool) error {
	if source == nil {
		return errors.New("dice source is nil")
	}
	if outcome == nil || !eligible || !outcome.Hit {
		return nil
	}
	count := 1
	if outcome.Crit {
		count = 2
	}
	damageType := "piercing"
	if len(outcome.Damage) > 0 && outcome.Damage[0].Type != "" {
		damageType = outcome.Damage[0].Type
	}
	damage, total, err := rollDamage(source, count, 6, 0, damageType, "sneak_attack")
	if err != nil {
		return err
	}
	outcome.Damage = append(outcome.Damage, damage...)
	outcome.Total += total
	outcome.HPAfter = max(0, outcome.HPBefore-outcome.Total)
	return nil
}

// Slam resolves the drowned thrall's +3, 1d8+1 bludgeoning attack.
func Slam(source *dice.Roller, attackID, target string, targetAC, hpBefore int, adv int8) (AttackOutcome, error) {
	return Attack(source, attackID, "thrall", target, 3, targetAC, "1d8", 1, "bludgeoning", hpBefore, adv)
}

// ApplyDamage changes HP and adds the status appropriate to the resulting HP.
func ApplyDamage(creature *CreatureState, amount int) {
	if creature == nil || amount < 0 {
		return
	}
	creature.HP = max(0, creature.HP-amount)
	if creature.HP == 0 {
		addCondition(creature, Defeated)
		addCondition(creature, Down)
		addCondition(creature, Prone)
	} else if creature.HP*2 <= creature.MaxHP {
		addCondition(creature, Bloodied)
	}
}

// EndCombat stabilizes a down PC at one HP, or marks a fleeing creature.
func EndCombat(creature *CreatureState, fled bool) {
	if creature == nil {
		return
	}
	if fled {
		addCondition(creature, Fled)
		return
	}
	if hasCondition(creature, Down) {
		creature.HP = 1
		removeCondition(creature, Down)
		removeCondition(creature, Prone)
		removeCondition(creature, Defeated)
	}
}

func rollDamage(source *dice.Roller, count, sides, bonus int, damageType, origin string) ([]DamageRoll, int, error) {
	faces := make([]int, count)
	total := bonus
	for i := range faces {
		draw, err := source.Roll(sides)
		if err != nil {
			return nil, 0, err
		}
		faces[i] = draw.Face
		total += draw.Face
	}
	return []DamageRoll{{Dice: strconv.Itoa(count) + "d" + strconv.Itoa(sides), Faces: faces, Bonus: bonus, Total: total, Type: damageType, Source: origin}}, total, nil
}

func parseDice(notation string) (int, int, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(notation)), "d")
	if len(parts) != 2 {
		return 0, 0, errors.New("damage dice must be NdM")
	}
	count, err := parsePositive(parts[0])
	if err != nil {
		return 0, 0, err
	}
	sides, err := parsePositive(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return count, sides, nil
}

func parsePositive(value string) (int, error) {
	n := 0
	if value == "" {
		return 0, errors.New("dice value is empty")
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, errors.New("dice value is invalid")
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, errors.New("dice value must be positive")
	}
	return n, nil
}
func addCondition(c *CreatureState, status Condition) {
	if !hasCondition(c, status) {
		c.Conditions = append(c.Conditions, status)
	}
}
func hasCondition(c *CreatureState, status Condition) bool {
	for _, current := range c.Conditions {
		if current == status {
			return true
		}
	}
	return false
}
func removeCondition(c *CreatureState, status Condition) {
	for i, current := range c.Conditions {
		if current == status {
			c.Conditions = append(c.Conditions[:i], c.Conditions[i+1:]...)
			return
		}
	}
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
