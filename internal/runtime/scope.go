package runtime

import (
	"context"
	"sort"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// ScopeTree owns cancellation contexts for the runtime's nested work scopes.
// A tree is safe for concurrent registration, cancellation, and inspection.
type ScopeTree struct {
	mu   sync.Mutex
	root *scopeNode
}

type scopeNode struct {
	scope    domain.Scope
	ctx      context.Context
	cancel   context.CancelFunc
	children map[scopeKey]*scopeNode
}

type scopeKey struct {
	machine string
	epoch   uint32
	key     string
}

// NewScopeTree constructs a tree rooted at parent. A nil parent is treated as
// context.Background so callers can safely use constructor-injected parents.
func NewScopeTree(parent context.Context) *ScopeTree {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	return &ScopeTree{root: &scopeNode{ctx: ctx, cancel: cancel, children: make(map[scopeKey]*scopeNode)}}
}

// Context returns the context for scope, creating it under parent when absent.
// A nil parent means the tree root. Reusing a scope returns its existing
// context; callers must use a new epoch for a new instance.
func (t *ScopeTree) Context(scope domain.Scope, parent domain.Scope) context.Context {
	node := t.ensure(scope, parent)
	return node.ctx
}

// Ensure creates or returns scope and its context. If scope already exists
// with a different epoch for the same machine and key, the old instance is
// cancelled before the new one is installed.
func (t *ScopeTree) Ensure(scope domain.Scope, parent domain.Scope) context.Context {
	return t.Context(scope, parent)
}

func (t *ScopeTree) ensure(scope, parent domain.Scope) *scopeNode {
	t.mu.Lock()
	defer t.mu.Unlock()
	if isRoot(scope) {
		return t.root
	}
	parentNode := t.nodeLocked(parent)
	if !isRoot(parent) && parentNode == t.root {
		parentNode = t.ensureChildLocked(t.root, parent)
	}
	key := makeScopeKey(scope)
	for childKey, child := range parentNode.children {
		if child.scope.Machine == scope.Machine && child.scope.Key == scope.Key && childKey != key {
			child.cancel()
			delete(parentNode.children, childKey)
		}
	}
	if existing := parentNode.children[key]; existing != nil {
		return existing
	}
	ctx, cancel := context.WithCancel(parentNode.ctx)
	node := &scopeNode{scope: scope, ctx: ctx, cancel: cancel, children: make(map[scopeKey]*scopeNode)}
	parentNode.children[key] = node
	return node
}

func (t *ScopeTree) ensureChildLocked(parentNode *scopeNode, scope domain.Scope) *scopeNode {
	key := makeScopeKey(scope)
	if existing := parentNode.children[key]; existing != nil {
		return existing
	}
	ctx, cancel := context.WithCancel(parentNode.ctx)
	node := &scopeNode{scope: scope, ctx: ctx, cancel: cancel, children: make(map[scopeKey]*scopeNode)}
	parentNode.children[key] = node
	return node
}

// Cancel cancels scope and all descendants. Cancelling an unknown scope is a
// no-op, which makes transitions idempotent.
func (t *ScopeTree) Cancel(scope domain.Scope) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if isRoot(scope) {
		t.root.cancel()
		return
	}
	parent, key, node := t.findLocked(scope)
	if node == nil {
		return
	}
	node.cancel()
	delete(parent.children, key)
}

// CancelKey cancels one keyed child while leaving sibling keys and the parent
// scope alive.
func (t *ScopeTree) CancelKey(scope domain.Scope) { t.Cancel(scope) }

// Close cancels every scope and releases the root context.
func (t *ScopeTree) Close() { t.Cancel(domain.Scope{}) }

// Scopes returns a stable, sorted snapshot of active non-root scopes.
func (t *ScopeTree) Scopes() []domain.ScopeState {
	t.mu.Lock()
	defer t.mu.Unlock()
	var states []domain.ScopeState
	collectScopes(t.root, &states)
	sort.Slice(states, func(i, j int) bool {
		if states[i].Scope.Machine != states[j].Scope.Machine {
			return states[i].Scope.Machine < states[j].Scope.Machine
		}
		if states[i].Scope.Key != states[j].Scope.Key {
			return states[i].Scope.Key < states[j].Scope.Key
		}
		return states[i].Scope.Epoch < states[j].Scope.Epoch
	})
	return states
}

func collectScopes(node *scopeNode, states *[]domain.ScopeState) {
	for _, child := range node.children {
		*states = append(*states, domain.ScopeState{Scope: child.scope, Epoch: child.scope.Epoch, State: "active"})
		collectScopes(child, states)
	}
}

func (t *ScopeTree) nodeLocked(scope domain.Scope) *scopeNode {
	if isRoot(scope) {
		return t.root
	}
	_, _, node := t.findLocked(scope)
	if node != nil {
		return node
	}
	return t.root
}

func (t *ScopeTree) findLocked(scope domain.Scope) (*scopeNode, scopeKey, *scopeNode) {
	key := makeScopeKey(scope)
	var find func(*scopeNode) (*scopeNode, *scopeNode)
	find = func(parent *scopeNode) (*scopeNode, *scopeNode) {
		if child := parent.children[key]; child != nil {
			return parent, child
		}
		for _, child := range parent.children {
			if parentNode, found := find(child); found != nil {
				return parentNode, found
			}
		}
		return nil, nil
	}
	parent, node := find(t.root)
	return parent, key, node
}

func makeScopeKey(scope domain.Scope) scopeKey {
	return scopeKey{machine: string(scope.Machine), epoch: scope.Epoch, key: scope.Key}
}

func isRoot(scope domain.Scope) bool { return scope.Machine == "" && scope.Key == "" }
