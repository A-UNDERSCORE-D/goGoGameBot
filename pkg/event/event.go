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

// BaseEvent implements parts of the Event interface in order to reduce boilerplate. Its expected that BaseEvent
// is embedded in simple implementations of Event
type BaseEvent struct {
	m         sync.RWMutex
	cancelled bool
	Name_     string
}

// IsCancelled returns the cancellation state of the BaseEvent
func (b *BaseEvent) IsCancelled() bool {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.cancelled
}

// SetCancelled sets the cancellation state of the BaseEvent
func (b *BaseEvent) SetCancelled(c bool) bool {
	b.m.Lock()
	old := b.cancelled
	b.cancelled = c
	b.m.Unlock()
	return old
}

// Name returns the name of the event this BaseEvent represents
func (b *BaseEvent) Name() string {
	return b.Name_
}

// SimpleEvent is a basic Event implementation, its useful to provide a notification but not pass any data
type SimpleEvent struct {
	BaseEvent
}

// EventType returns the type of this event, For the Default event, this is always "Default"
func (SimpleEvent) EventType() string { return "SimpleEvent" }

// NewSimpleEvent creates a new SimpleEvent and sets its name and Argmap to the provided values
func NewSimpleEvent(name string) *SimpleEvent {
	return &SimpleEvent{BaseEvent: BaseEvent{Name_: name}}
}
