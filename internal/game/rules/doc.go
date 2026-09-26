// Package rules contains the pure rules subset used by DungeonFlux combat and
// character creation. BuildHero and Classes expose the level-one templates;
// Attack, ApplySneakAttack, ApplyDamage, and EndCombat are the combat surface
// consumed by the combat machine. AttackOutcome.Damage identifies weapon and
// sneak_attack components, while Condition values identify bloodied, down,
// defeated, prone, and fled states.
package rules
