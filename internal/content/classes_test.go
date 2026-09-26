package content

import "testing"

func TestClasses_AllHaveEnglishAndSpanishCopy(t *testing.T) {
	for _, classID := range Classes() {
		for _, locale := range []string{"en", "es"} {
			copy, ok := ClassCopyFor(locale, classID)
			if !ok || copy.Name == "" || copy.Description == "" {
				t.Errorf("%s missing %s copy: %#v", classID, locale, copy)
			}
		}
	}
}

func TestClasses_ReturnsIndependentOrder(t *testing.T) {
	classes := Classes()
	if len(classes) != 12 || classes[0] != ClassBarbarian || classes[len(classes)-1] != ClassWizard {
		t.Fatalf("unexpected class order: %v", classes)
	}
	classes[0] = ClassWizard
	if Classes()[0] != ClassBarbarian {
		t.Fatal("Classes returned mutable package state")
	}
}

func TestClassCopyFor_LocaleFallback(t *testing.T) {
	es, ok := ClassCopyFor("es-MX", ClassPaladin)
	if !ok || es.Name != "Paladín" {
		t.Fatalf("es fallback failed: %#v, %v", es, ok)
	}
	en, ok := ClassCopyFor("fr", ClassPaladin)
	if !ok || en.Name != "Paladin" {
		t.Fatalf("english fallback failed: %#v, %v", en, ok)
	}
	if _, ok := ClassCopyFor("en", ClassID("missing")); ok {
		t.Fatal("unknown class unexpectedly resolved")
	}
}

func TestClassMoveCopy_Localized(t *testing.T) {
	if label, reason := ClassMoveCopy("en"); label != ClassMoveLabel || reason != ClassMoveReason {
		t.Fatalf("english copy mismatch: %q / %q", label, reason)
	}
	label, reason := ClassMoveCopy("es")
	if label != "Elige una clase" || reason == "" {
		t.Fatalf("spanish copy mismatch: %q / %q", label, reason)
	}
}
