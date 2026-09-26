package content

// ClassID is the stable, lowercase identifier used by the class creation
// move and by the localized class catalog.
type ClassID string

const (
	ClassBarbarian ClassID = "barbarian"
	ClassBard      ClassID = "bard"
	ClassCleric    ClassID = "cleric"
	ClassDruid     ClassID = "druid"
	ClassFighter   ClassID = "fighter"
	ClassMonk      ClassID = "monk"
	ClassPaladin   ClassID = "paladin"
	ClassRanger    ClassID = "ranger"
	ClassRogue     ClassID = "rogue"
	ClassSorcerer  ClassID = "sorcerer"
	ClassWarlock   ClassID = "warlock"
	ClassWizard    ClassID = "wizard"
)

// ClassCopy contains the localized display copy for one class.
type ClassCopy struct {
	Name        string
	Description string
}

// ClassMoveLabel is the English fallback label for the class picker.
const ClassMoveLabel = "Choose a class"

// ClassMoveReason is the English fallback reason shown before a class is set.
const ClassMoveReason = "Choose a class before rolling"

var classIDs = [...]ClassID{
	ClassBarbarian, ClassBard, ClassCleric, ClassDruid, ClassFighter, ClassMonk,
	ClassPaladin, ClassRanger, ClassRogue, ClassSorcerer, ClassWarlock, ClassWizard,
}

var classCopy = map[string]map[ClassID]ClassCopy{
	"en": {
		ClassBarbarian: {"Barbarian", "A fierce front-line warrior who turns fury into force."},
		ClassBard:      {"Bard", "A charismatic storyteller who inspires allies and outsmarts danger."},
		ClassCleric:    {"Cleric", "A devoted champion who channels sacred power to protect the party."},
		ClassDruid:     {"Druid", "A nature-bound spellcaster who adapts to every wild challenge."},
		ClassFighter:   {"Fighter", "A disciplined weapon master who stands ready for any fight."},
		ClassMonk:      {"Monk", "A swift martial artist who turns focus and movement into defense."},
		ClassPaladin:   {"Paladin", "A steadfast champion whose oath makes every strike count."},
		ClassRanger:    {"Ranger", "A wilderness scout who combines sharp aim with practical magic."},
		ClassRogue:     {"Rogue", "A clever skirmisher who finds the opening and strikes where it hurts."},
		ClassSorcerer:  {"Sorcerer", "A natural spellcaster who shapes powerful magic from within."},
		ClassWarlock:   {"Warlock", "A pact-bound spellcaster who bargains for uncanny power."},
		ClassWizard:    {"Wizard", "A learned spellcaster whose careful preparation solves impossible problems."},
	},
	"es": {
		ClassBarbarian: {"Bárbaro", "Un guerrero de primera línea que convierte la furia en fuerza."},
		ClassBard:      {"Bardo", "Un narrador carismático que inspira aliados y supera peligros."},
		ClassCleric:    {"Clérigo", "Un campeón devoto que canaliza poder sagrado para proteger al grupo."},
		ClassDruid:     {"Druida", "Un lanzador ligado a la naturaleza que se adapta a cada desafío salvaje."},
		ClassFighter:   {"Guerrero", "Un maestro disciplinado de las armas, preparado para cualquier combate."},
		ClassMonk:      {"Monje", "Un artista marcial veloz que convierte concentración y movimiento en defensa."},
		ClassPaladin:   {"Paladín", "Un campeón firme cuyo juramento hace que cada golpe cuente."},
		ClassRanger:    {"Explorador", "Un explorador de la naturaleza que combina puntería y magia práctica."},
		ClassRogue:     {"Pícaro", "Un escaramuzador astuto que encuentra el hueco y golpea donde duele."},
		ClassSorcerer:  {"Hechicero", "Un lanzador innato que moldea la poderosa magia que lleva dentro."},
		ClassWarlock:   {"Brujo", "Un lanzador ligado por un pacto que negocia por poder sobrenatural."},
		ClassWizard:    {"Mago", "Un lanzador estudioso cuya preparación resuelve problemas imposibles."},
	},
}

// Classes returns all selectable classes in the stable picker order.
func Classes() []ClassID {
	return append([]ClassID(nil), classIDs[:]...)
}

// ClassCopyFor returns localized copy for a class, falling back to English.
func ClassCopyFor(locale string, classID ClassID) (ClassCopy, bool) {
	locale = normalizeLocale(locale)
	copy, ok := classCopy[locale][classID]
	if ok {
		return copy, true
	}
	copy, ok = classCopy["en"][classID]
	return copy, ok
}

// ClassMoveCopy returns the localized class-picker label and disabled reason.
func ClassMoveCopy(locale string) (label, reason string) {
	if normalizeLocale(locale) == "es" {
		return "Elige una clase", "Elige una clase antes de crear tu héroe"
	}
	return ClassMoveLabel, ClassMoveReason
}

func normalizeLocale(locale string) string {
	if len(locale) >= 2 && (locale[0] == 'e' || locale[0] == 'E') && (locale[1] == 's' || locale[1] == 'S') {
		return "es"
	}
	return "en"
}
