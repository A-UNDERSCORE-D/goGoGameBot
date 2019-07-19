package event

import "sync"

// Event represents any event that can be fired over the event bus
type Event interface {
	// Name refers to the Name of the specific event this Event represents. It should be based on the data within the Event
	Name() string
	// EventType refers to the type of this Event, its should be static for all instances of an implementation of Event
	EventType() string
	// IsCancelled returns whether or not this Event has been Cancelled
	IsCancelled() bool
	// SetCancelled sets the cancelled state of the Event
	SetCancelled(bool) bool
	// TODO: ID?
}

// DefaultEvent is the base event implementation
type DefaultEvent struct {
	m         *sync.RWMutex
	cancelled bool
	ArgMap    ArgMap
	name      string
}

// EventType returns the type of this event, For the Default event, this is always "Default"
func (DefaultEvent) EventType() string { return "Default" }

// Name Returns the name of this specific version of DefaultEvent
func (d *DefaultEvent) Name() string { return d.name }

// IsCancelled returns the cancellation state of the event
func (d *DefaultEvent) IsCancelled() bool {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.cancelled
}

// SetCancelled sets the cancellation state of the event
func (d *DefaultEvent) SetCancelled(toSet bool) bool {
	d.m.Lock()
	defer d.m.Unlock()
	old := d.cancelled
	d.cancelled = toSet
	return old
}

// NewDefaultEvent creates a new DefaultEvent and sets its name and Argmap to the provided values
func NewDefaultEvent(name string, argMap ArgMap) *DefaultEvent {
	return &DefaultEvent{name: name, ArgMap: argMap}
}
