package fsm

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Scope identifies one running machine instance and its per-instance key.
type Scope struct {
	Machine vocab.MachineID
	Key     string
	Epoch   uint32
}

// ScopeID addresses a scope without including its current epoch.
type ScopeID struct {
	Machine vocab.MachineID
	Key     string
}

// ScopeError reports an invalid scope-tree operation.
type ScopeError struct {
	ID     ScopeID
	Reason string
}

func (e *ScopeError) Error() string {
	return fmt.Sprintf("scope %q/%q: %s", e.ID.Machine, e.ID.Key, e.Reason)
}

const (
	scopeMissing = "missing"
	scopeExists  = "already_exists"
	badParent    = "parent_missing"
)

type scopeNode struct {
	scope    Scope
	parent   ScopeID
	children map[ScopeID]struct{}
}

// Scopes owns a tree of machine instances. It is intended to be held by the
// single engine loop; it has no locking or background work.
type Scopes struct {
	nodes map[ScopeID]*scopeNode
}

// NewScopes creates an empty scope tree.
func NewScopes() Scopes {
	return Scopes{nodes: make(map[ScopeID]*scopeNode)}
}

// Open creates a root scope when parent is nil, or a child below parent.
func (s *Scopes) Open(machine vocab.MachineID, key string, parent *ScopeID) (Scope, error) {
	if machine == "" {
		return Scope{}, &ScopeError{Reason: scopeMissing}
	}
	id := ScopeID{Machine: machine, Key: key}
	if _, exists := s.nodes[id]; exists {
		return Scope{}, &ScopeError{ID: id, Reason: scopeExists}
	}
	parentID := ScopeID{}
	if parent != nil {
		parentID = *parent
		if _, exists := s.nodes[parentID]; !exists {
			return Scope{}, &ScopeError{ID: parentID, Reason: badParent}
		}
	}
	node := &scopeNode{scope: Scope{Machine: machine, Key: key}, parent: parentID, children: make(map[ScopeID]struct{})}
	s.nodes[id] = node
	if parent != nil {
		s.nodes[parentID].children[id] = struct{}{}
	}
	return node.scope, nil
}

// Get returns the current scope value for an ID.
func (s *Scopes) Get(id ScopeID) (Scope, bool) {
	node, exists := s.nodes[id]
	if !exists {
		return Scope{}, false
	}
	return node.scope, true
}

// Advance increments a scope epoch for a non-internal transition. Internal
// transitions preserve the epoch so work remains valid across self-transitions.
func (s *Scopes) Advance(id ScopeID, internal bool) (Scope, error) {
	node, exists := s.nodes[id]
	if !exists {
		return Scope{}, &ScopeError{ID: id, Reason: scopeMissing}
	}
	if !internal {
		node.scope.Epoch++
	}
	return node.scope, nil
}

// Current reports whether an event scope still names the live epoch.
func (s *Scopes) Current(scope Scope) bool {
	current, exists := s.Get(ScopeID{Machine: scope.Machine, Key: scope.Key})
	return exists && current.Epoch == scope.Epoch
}

// Cancel removes a scope and every descendant, making their late events stale.
func (s *Scopes) Cancel(id ScopeID) error {
	node, exists := s.nodes[id]
	if !exists {
		return &ScopeError{ID: id, Reason: scopeMissing}
	}
	for child := range node.children {
		if err := s.Cancel(child); err != nil {
			return err
		}
	}
	if node.parent != (ScopeID{}) {
		delete(s.nodes[node.parent].children, id)
	}
	delete(s.nodes, id)
	return nil
}
