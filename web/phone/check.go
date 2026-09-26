package phone

import "strings"

// CheckPresentation is the stable copy and outcome data for a phone check.
type CheckPresentation struct {
	// Name is the display title for the check.
	Name string
	// Ability is the governing ability label.
	Ability string
	// Skill is the proficiency label used by the check.
	Skill string
	// Description is the offer copy shown before the roll.
	Description string
	// Quote is the italic prompt shown on the result screen.
	Quote string
	// ResultText is the explanatory resolution copy.
	ResultText string
	// Outcome is the success or failure banner label.
	Outcome string
	// Total is the resolved check total.
	Total int32
	// DC is the target number for the check.
	DC int32
	// D20 is the kept die face.
	D20 int32
	// Modifier is the signed ability and proficiency bonus.
	Modifier int32
	// HasRoll reports whether a valid d20 face is available.
	HasRoll bool
	// Success reports the resolved outcome when HasOutcome is true.
	Success bool
	// HasOutcome reports whether success or failure is known.
	HasOutcome bool
}

// CheckPresentationFromDice turns the wire-safe dice snapshot into screen copy.
func CheckPresentationFromDice(snapshot DiceSnapshot) CheckPresentation {
	presentation := CheckPresentation{
		Name:        "Persuasion Check",
		Ability:     "Charisma",
		Skill:       "Persuasion",
		Description: "You try to reason with Marra, appealing to her better nature.",
		Quote:       "You gather your thoughts and choose your words carefully…",
		ResultText:  strings.TrimSpace(snapshot.Outcome),
		DC:          snapshot.DC,
		D20:         snapshot.D20,
		Modifier:    snapshot.Modifier,
	}
	if presentation.DC <= 0 {
		presentation.DC = 10
	}
	presentation.HasRoll = snapshot.D20 >= 1 && snapshot.D20 <= 20
	presentation.Total = snapshot.Total
	if presentation.HasRoll && presentation.Total == 0 {
		presentation.Total = snapshot.D20 + snapshot.Modifier
	}
	presentation.HasOutcome, presentation.Success = checkOutcome(snapshot)
	if presentation.HasOutcome {
		if presentation.Success {
			presentation.Outcome = "Success"
		} else {
			presentation.Outcome = "Failure"
		}
	}
	if presentation.ResultText == "" || strings.EqualFold(presentation.ResultText, presentation.Outcome) {
		presentation.ResultText = checkResultText(presentation)
	}
	return presentation
}

func checkOutcome(snapshot DiceSnapshot) (bool, bool) {
	outcome := strings.ToLower(strings.TrimSpace(snapshot.Outcome))
	if strings.Contains(outcome, "success") || strings.Contains(outcome, "succeed") || strings.Contains(outcome, "victory") {
		return true, true
	}
	if strings.Contains(outcome, "failure") || strings.Contains(outcome, "fail") || strings.Contains(outcome, "refuse") {
		return true, false
	}
	if snapshot.Phase == DiceResolved && snapshot.D20 >= 1 && snapshot.D20 <= 20 {
		return true, snapshot.D20+snapshot.Modifier >= snapshot.DC
	}
	return false, false
}

func checkResultText(presentation CheckPresentation) string {
	if presentation.HasOutcome && presentation.Success {
		return "Marra hesitates, then lowers her voice. Fine. He was taken last night, down by the eastern docks…"
	}
	if presentation.HasOutcome {
		return "Marra folds her arms. Whatever you said, she is not ready to trust you with more."
	}
	if presentation.D20 > 0 {
		return "The die settles while the table waits for its meaning."
	}
	return presentation.Description
}

// SignedModifier formats an ability modifier for the check panel.
func SignedModifier(modifier int32) string {
	if modifier >= 0 {
		return "+" + itoa(modifier)
	}
	return "-" + itoa(-modifier)
}

func itoa(value int32) string {
	if value == 0 {
		return "0"
	}
	const digits = "0123456789"
	var reversed [12]byte
	index := len(reversed)
	for value > 0 {
		index--
		reversed[index] = digits[value%10]
		value /= 10
	}
	return string(reversed[index:])
}
