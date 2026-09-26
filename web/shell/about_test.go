package main

import (
	"strings"
	"testing"
)

func TestAboutAttribution_ContainsRequiredNotice(t *testing.T) {
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

func TestAboutAttribution_DescribesModifiedThrall(t *testing.T) {
	for _, want := range []string{"drowned thrall", "Zombie", "fixed at 12", "Undead Fortitude is removed"} {
		if !strings.Contains(DrownedThrallNotice, want) {
			t.Errorf("DrownedThrallNotice does not contain %q", want)
		}
	}
}

func TestRouteAbout_IsAboutPath(t *testing.T) {
	if got := string(RouteAbout); got != "/about" {
		t.Fatalf("RouteAbout = %q, want /about", got)
	}
}
