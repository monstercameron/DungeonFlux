package phone

import "testing"

func TestFrameTabs_KeepFiveDestinationsAndRaisedMode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		active PhoneTabID
		mode   PhoneMode
		icon   string
	}{
		{name: "conversation", active: PhoneTabPlay, mode: PhoneModePlay, icon: "●"},
		{name: "exploration", active: PhoneTabMap, mode: PhoneModeExplore, icon: "✥"},
		{name: "combat", active: PhoneTabPlay, mode: PhoneModeCombat, icon: "⚔"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tabs := FrameTabs(tc.active, tc.mode)
			if len(tabs) != 5 || tabs[2].ID != PhoneTabPlay || !tabs[2].Raised || tabs[2].Icon != tc.icon {
				t.Fatalf("tabs = %+v", tabs)
			}
			active := 0
			for _, tab := range tabs {
				if tab.Active {
					active++
				}
			}
			if active != 1 {
				t.Fatalf("active tabs = %d", active)
			}
		})
	}
}

func TestPhoneComponentHelpers_ClampAndDefault(t *testing.T) {
	cases := []struct {
		name string
		hp   int32
		max  int32
		want int32
	}{
		{name: "empty", hp: 0, max: 10, want: 0},
		{name: "negative", hp: -2, max: 10, want: 0},
		{name: "partial", hp: 5, max: 10, want: 50},
		{name: "over", hp: 12, max: 10, want: 100},
		{name: "missing max", hp: 5, max: 0, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HPPercent(tc.hp, tc.max); got != tc.want {
				t.Fatalf("HPPercent(%d, %d) = %d, want %d", tc.hp, tc.max, got, tc.want)
			}
		})
	}
	if got := NormalizePhoneMode("unexpected"); got != PhoneModePlay || ModeLabel("unexpected") != "Play" {
		t.Fatalf("unknown mode fallback = %q / %q", got, ModeLabel("unexpected"))
	}
	if got := ChoiceRowReason(ChoiceRowModel{}); got != "Unavailable right now" {
		t.Fatalf("default reason = %q", got)
	}
}
