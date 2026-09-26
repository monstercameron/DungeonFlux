package dm

// EndCardModel contains the fixed copy shown after the cliffhanger line.
type EndCardModel struct {
	Title       string
	Subtitle    string
	Attribution string
}

// SRDAttribution is the required SRD 5.2.1 attribution statement.
const SRDAttribution = `This work includes material from the System Reference Document 5.2.1 ("SRD 5.2.1") by Wizards of the Coast LLC, available at https://www.dndbeyond.com/srd. The SRD 5.2.1 is licensed under the Creative Commons Attribution 4.0 International License, available at https://creativecommons.org/licenses/by/4.0/legalcode.`

// NewEndCardModel creates the end-card copy for the terminal phase state.
func NewEndCardModel() EndCardModel {
	return EndCardModel{
		Title:       "The bell remembers.",
		Subtitle:    "Thank you for playing DungeonFlux.",
		Attribution: SRDAttribution,
	}
}

// EndCardReady reports whether the model has all copy needed to render.
func EndCardReady(model EndCardModel) bool {
	return model.Title != "" && model.Subtitle != "" && model.Attribution == SRDAttribution
}
