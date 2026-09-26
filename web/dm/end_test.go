package dm

import (
	"strings"
	"testing"
)

func TestNewEndCardModel_ContainsTerminalCopyAndAttribution(t *testing.T) {
	model := NewEndCardModel()
	if model.Title == "" || model.Subtitle == "" {
		t.Fatalf("end card copy = %+v", model)
	}
	if model.Attribution != SRDAttribution {
		t.Fatal("end card must use the canonical SRD attribution")
	}
}

func TestEndCardReady_RequiresCanonicalAttribution(t *testing.T) {
	base := NewEndCardModel()
	tests := []struct {
		name  string
		model EndCardModel
		want  bool
	}{
		{name: "complete", model: base, want: true},
		{name: "missing title", model: EndCardModel{Subtitle: base.Subtitle, Attribution: base.Attribution}},
		{name: "missing subtitle", model: EndCardModel{Title: base.Title, Attribution: base.Attribution}},
		{name: "altered attribution", model: EndCardModel{Title: base.Title, Subtitle: base.Subtitle, Attribution: strings.TrimSuffix(base.Attribution, ".")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EndCardReady(test.model); got != test.want {
				t.Fatalf("EndCardReady() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSRDAttribution_ContainsRequiredLicenseDetails(t *testing.T) {
	for _, want := range []string{
		"System Reference Document 5.2.1",
		"Wizards of the Coast LLC",
		"https://www.dndbeyond.com/srd",
		"Creative Commons Attribution 4.0 International License",
		"https://creativecommons.org/licenses/by/4.0/legalcode",
	} {
		if !strings.Contains(SRDAttribution, want) {
			t.Errorf("SRDAttribution does not contain %q", want)
		}
	}
}
