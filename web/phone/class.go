package phone

import (
	"errors"
	"strings"
)

// ClassOption describes one selectable SRD class for the phone picker.
type ClassOption struct {
	// ID is the lowercase class identifier sent to the engine.
	ID string
	// Label is the localized class name.
	Label string
	// Role is the localized one-line role description.
	Role string
	// Crest is the decorative class crest shown beside the name.
	Crest string
}

type classCopy struct {
	id, name, role string
	esName, esRole string
	crest          string
}

var creationClassCopies = [...]classCopy{
	{"barbarian", "Barbarian", "A fierce front-line warrior who turns fury into force.", "Bárbaro", "Un guerrero de primera línea que convierte la furia en fuerza.", "⚔"},
	{"bard", "Bard", "A charismatic storyteller who inspires allies and outsmarts danger.", "Bardo", "Un narrador carismático que inspira aliados y supera peligros.", "♫"},
	{"cleric", "Cleric", "A devoted champion who channels sacred power to protect the party.", "Clérigo", "Un campeón devoto que canaliza poder sagrado para proteger al grupo.", "✚"},
	{"druid", "Druid", "A nature-bound spellcaster who adapts to every wild challenge.", "Druida", "Un lanzador ligado a la naturaleza que se adapta a cada desafío salvaje.", "❧"},
	{"fighter", "Fighter", "A disciplined weapon master who stands ready for any fight.", "Guerrero", "Un maestro disciplinado de las armas, preparado para cualquier combate.", "◈"},
	{"monk", "Monk", "A swift martial artist who turns focus and movement into defense.", "Monje", "Un artista marcial veloz que convierte concentración y movimiento en defensa.", "☯"},
	{"paladin", "Paladin", "A steadfast champion whose oath makes every strike count.", "Paladín", "Un campeón firme cuyo juramento hace que cada golpe cuente.", "✦"},
	{"ranger", "Ranger", "A wilderness scout who combines sharp aim with practical magic.", "Explorador", "Un explorador de la naturaleza que combina puntería y magia práctica.", "⌁"},
	{"rogue", "Rogue", "A clever skirmisher who finds the opening and strikes where it hurts.", "Pícaro", "Un escaramuzador astuto que encuentra el hueco y golpea donde duele.", "◒"},
	{"sorcerer", "Sorcerer", "A natural spellcaster who shapes powerful magic from within.", "Hechicero", "Un lanzador innato que moldea la poderosa magia que lleva dentro.", "✧"},
	{"warlock", "Warlock", "A pact-bound spellcaster who bargains for uncanny power.", "Brujo", "Un lanzador ligado por un pacto que negocia por poder sobrenatural.", "☽"},
	{"wizard", "Wizard", "A learned spellcaster whose careful preparation solves impossible problems.", "Mago", "Un lanzador estudioso cuya preparación resuelve problemas imposibles.", "✺"},
}

// CreationClasses returns the twelve legal SRD classes in stable display order.
func CreationClasses() []ClassOption { return CreationClassesForLocale("en") }

// CreationClassesForLocale returns localized class labels and role copy.
func CreationClassesForLocale(locale string) []ClassOption {
	es := strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "es")
	classes := make([]ClassOption, 0, len(creationClassCopies))
	for _, item := range creationClassCopies {
		name, role := item.name, item.role
		if es {
			name, role = item.esName, item.esRole
		}
		classes = append(classes, ClassOption{ID: item.id, Label: name, Role: role, Crest: item.crest})
	}
	return classes
}

// SelectClass records a legal class selection.
func (m *CreationModel) SelectClass(class string) error {
	if m == nil {
		return errors.New("creation model is unavailable")
	}
	class = strings.ToLower(strings.TrimSpace(class))
	for _, item := range creationClassCopies {
		if class == item.id {
			m.state.Class, m.state.Error = class, ""
			return nil
		}
	}
	return m.selectValue(&m.state.Class, class, nil, "class")
}
