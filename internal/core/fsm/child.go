package fsm

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	// EventChildDone is the event routed when a child completes successfully.
	EventChildDone Event = "child_done"
	// EventChildFailed is the event routed when a child fails.
	EventChildFailed Event = "child_failed"
)

// ChildResult reports a terminal result from a child machine.
type ChildResult struct {
	Child  ScopeID
	Failed bool
	Event  Event
	Value  any
}

// RoutedEvent is the parent-facing event produced from a child result.
type RoutedEvent struct {
	Target ScopeID
	Source ScopeID
	Event  Event
	Value  any
}

// Child describes a child machine and its parent pointer.
type Child struct {
	ID      ScopeID
	Parent  ScopeID
	Machine vocab.MachineID
}

// ChildError reports an invalid child registration or result.
type ChildError struct {
	Child  ScopeID
	Reason string
}

func (e *ChildError) Error() string {
	return fmt.Sprintf("child %q/%q: %s", e.Child.Machine, e.Child.Key, e.Reason)
}

const (
	childExists  = "already_exists"
	childMissing = "missing"
	childParent  = "parent_missing"
)

// Children tracks child machines and routes their terminal results.
type Children struct {
	entries map[ScopeID]Child
}

// NewChildren creates an empty child registry.
func NewChildren() Children {
	return Children{entries: make(map[ScopeID]Child)}
}

// Add registers a child with its parent pointer.
func (c *Children) Add(child ScopeID, parent ScopeID, machine vocab.MachineID) error {
	if child.Machine == "" || parent.Machine == "" || machine == "" {
		return &ChildError{Child: child, Reason: childParent}
	}
	if _, exists := c.entries[child]; exists {
		return &ChildError{Child: child, Reason: childExists}
	}
	c.entries[child] = Child{ID: child, Parent: parent, Machine: machine}
	return nil
}

// Get returns a registered child and its parent pointer.
func (c *Children) Get(id ScopeID) (Child, bool) {
	child, exists := c.entries[id]
	return child, exists
}

// Route converts a child terminal result into an event for its parent. A
// supplied result event is retained; otherwise a standard done/failed event
// is selected.
func (c *Children) Route(result ChildResult) (RoutedEvent, error) {
	child, exists := c.entries[result.Child]
	if !exists {
		return RoutedEvent{}, &ChildError{Child: result.Child, Reason: childMissing}
	}
	event := result.Event
	if event == "" {
		event = EventChildDone
		if result.Failed {
			event = EventChildFailed
		}
	}
	return RoutedEvent{Target: child.Parent, Source: child.ID, Event: event, Value: result.Value}, nil
}

// Remove forgets a child so late completions can no longer route.
func (c *Children) Remove(id ScopeID) error {
	if _, exists := c.entries[id]; !exists {
		return &ChildError{Child: id, Reason: childMissing}
	}
	delete(c.entries, id)
	return nil
}
