package fsm

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestScopes_nestedInstancesAndEpochs(t *testing.T) {
	scopes := NewScopes()
	run, err := scopes.Open(vocab.MachineRun, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	runID := ScopeID{Machine: run.Machine, Key: run.Key}
	session, err := scopes.Open(vocab.MachineSession, "", &runID)
	if err != nil {
		t.Fatal(err)
	}
	sessionID := ScopeID{Machine: session.Machine, Key: session.Key}
	check, err := scopes.Open(vocab.MachineCheck, "1", &runID)
	if err != nil {
		t.Fatal(err)
	}
	if check.Epoch != 0 || !scopes.Current(check) {
		t.Fatalf("unexpected check scope: %#v", check)
	}
	combat, err := scopes.Open(vocab.MachineCombat, "1", &sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scopes.Open(vocab.MachinePTT, "1", &sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := scopes.Advance(ScopeID{Machine: combat.Machine, Key: combat.Key}, false); err != nil {
		t.Fatal(err)
	}
	if !scopes.Current(Scope{Machine: combat.Machine, Key: combat.Key, Epoch: 1}) {
		t.Fatal("advanced combat scope should accept its new epoch")
	}
}

func TestScopes_advanceSelfTransitionPreservesEpoch(t *testing.T) {
	scopes := NewScopes()
	scope, err := scopes.Open(vocab.MachineSession, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	id := ScopeID{Machine: scope.Machine, Key: scope.Key}
	if _, err := scopes.Advance(id, false); err != nil {
		t.Fatal(err)
	}
	updated, err := scopes.Advance(id, true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Epoch != 1 || !scopes.Current(Scope{Machine: updated.Machine, Epoch: 1}) {
		t.Fatalf("self-transition reset epoch: %#v", updated)
	}
	if scopes.Current(Scope{Machine: updated.Machine, Epoch: 0}) {
		t.Fatal("stale epoch accepted")
	}
}

func TestScopes_cancelDropsDescendantsButKeepsSiblings(t *testing.T) {
	scopes := NewScopes()
	run, err := scopes.Open(vocab.MachineRun, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	runID := ScopeID{Machine: run.Machine}
	phase, err := scopes.Open(vocab.MachineSession, "", &runID)
	if err != nil {
		t.Fatal(err)
	}
	phaseID := ScopeID{Machine: phase.Machine}
	utterance, err := scopes.Open(vocab.MachinePTT, "2", &phaseID)
	if err != nil {
		t.Fatal(err)
	}
	check, err := scopes.Open(vocab.MachineCheck, "2", &runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := scopes.Cancel(phaseID); err != nil {
		t.Fatal(err)
	}
	if scopes.Current(utterance) {
		t.Fatal("cancelled child accepted")
	}
	if !scopes.Current(check) {
		t.Fatal("sibling was cancelled")
	}
	if _, ok := scopes.Get(phaseID); ok {
		t.Fatal("cancelled phase remains")
	}
}

func TestScopes_rejectsMissingParentAndCancel(t *testing.T) {
	scopes := NewScopes()
	missing := ScopeID{Machine: vocab.MachineRun}
	_, err := scopes.Open(vocab.MachineCheck, "1", &missing)
	var scopeErr *ScopeError
	if !errors.As(err, &scopeErr) || scopeErr.Reason != badParent {
		t.Fatalf("unexpected parent error: %v", err)
	}
	if err := scopes.Cancel(missing); !errors.As(err, &scopeErr) || scopeErr.Reason != scopeMissing {
		t.Fatalf("unexpected cancel error: %v", err)
	}
}
