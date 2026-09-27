package phone

// PhoneTheme contains the visual tokens shared by every phone screen.
// Values are intentionally self-contained so the phone works without a font
// or stylesheet request to the network.
type PhoneTheme struct {
	Ink          string
	InkDeep      string
	Panel        string
	PanelRaised  string
	Parchment    string
	Muted        string
	Gold         string
	GoldBright   string
	Teal         string
	Blood        string
	BorderRadius string
	TouchTarget  string
	Transition   string
	Serif        string
	Sans         string
	PhoneMax     string
	HeaderHeight string
	TabBarHeight string
}

// DefaultPhoneTheme returns the dark-fantasy phone palette and layout tokens.
func DefaultPhoneTheme() PhoneTheme {
	return PhoneTheme{
		Ink: "#0f1117", InkDeep: "#0b0f16", Panel: "#171a23", PanelRaised: "#1d212c",
		Parchment: "#efe6d2", Muted: "#a89f8c", Gold: "#d9a441", GoldBright: "#e7c27a",
		Teal: "#2fb59a", Blood: "#b3372f", BorderRadius: "12px",
		TouchTarget: "48px", Transition: "180ms cubic-bezier(0.45, 0, 0.55, 1)",
		Serif:    "Cormorant Garamond, Cinzel, Georgia, serif",
		Sans:     "Inter, ui-sans-serif, system-ui, sans-serif",
		PhoneMax: "480px", HeaderHeight: "64px", TabBarHeight: "76px",
	}
}
