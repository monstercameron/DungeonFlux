package vocab

// EffectGenerateCharacterReference requests a multi-angle character reference
// sheet and its four cropped angle assets.
const EffectGenerateCharacterReference EffectKind = "generate_character_reference"

// ReferenceAngle identifies one view in a character turnaround.
type ReferenceAngle string

const (
	// ReferenceFront is the front-facing crop.
	ReferenceFront ReferenceAngle = "front"
	// ReferenceThreeQuarter is the three-quarter crop.
	ReferenceThreeQuarter ReferenceAngle = "three_quarter"
	// ReferenceSide is the side-profile crop.
	ReferenceSide ReferenceAngle = "side"
	// ReferenceBack is the rear-facing crop.
	ReferenceBack ReferenceAngle = "back"
)

// ReferenceAngles returns the canonical order used by the media pipeline.
func ReferenceAngles() []ReferenceAngle {
	return []ReferenceAngle{ReferenceFront, ReferenceThreeQuarter, ReferenceSide, ReferenceBack}
}
