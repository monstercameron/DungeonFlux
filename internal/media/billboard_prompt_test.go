package media

import (
	"strings"
	"testing"
)

func TestBillboardActions_WalkIsSupported(t *testing.T) {
	if !BillboardActions(BillboardWalk) {
		t.Fatal("walk action is not supported")
	}
	spec := BillboardSpec{Action: BillboardWalk, Subject: BillboardSubject{Look: "a paladin"}, LevelStill: []byte("level")}
	prompt := BillboardPrompt(spec)
	if !strings.Contains(prompt, "faces the drowned thrall") || !strings.Contains(prompt, "two deliberate in-place walking steps") {
		t.Fatalf("walk prompt does not stage the villain-facing movement: %q", prompt)
	}
}

func TestBillboardPrompt_ThrallFacesHeroes(t *testing.T) {
	spec := BillboardSpec{Action: BillboardIdle, Subject: BillboardSubject{Look: "drowned thrall", Weapon: "bell chain"}, LevelStill: []byte("level")}
	if prompt := BillboardPrompt(spec); !strings.Contains(prompt, "faces the heroes on the left side") {
		t.Fatalf("enemy prompt points at the wrong side: %q", prompt)
	}
}

func TestBillboardPrompt_AttackAndIdleFaceThrallAndLoop(t *testing.T) {
	for _, action := range []string{BillboardIdle, BillboardAttack} {
		t.Run(action, func(t *testing.T) {
			spec := BillboardSpec{Action: action, Subject: BillboardSubject{Look: "a hero", Weapon: "longsword"}, LevelStill: []byte("level")}
			prompt := BillboardPrompt(spec)
			if !strings.Contains(prompt, "faces the drowned thrall") {
				t.Fatalf("prompt lost villain-facing direction: %q", prompt)
			}
			if action == BillboardIdle && !strings.Contains(prompt, "same pose it started") {
				t.Fatalf("idle prompt is not loop-stable: %q", prompt)
			}
		})
	}
}
