package event

import (
	"sort"
	"sync"
)

// This is a reimplementation of eventmgr found at github.com/goshuirc/eventmgr with an ID system added.
// The original idea is theirs.

// Priority levels
const (
	PriHighest = 16
	PriHigh    = 32
	PriNorm    = 48
	PriLow     = 64
	PriLowest  = 80
)

// HandlerList is a slice of handlers with functions added to allow the slice to be sorted
type HandlerList []Handler

func (h HandlerList) Len() int {
	return len(h)
}

func (h HandlerList) Less(i, j int) bool {
	return h[i].Priority < h[j].Priority
}

func (h HandlerList) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Map is a map of string to HandlerList, it exists as a type alias for ease of use
type Map map[string]HandlerList

// ArgMap is a map of string to interface, it exists as a type alias for ease of use
type ArgMap map[string]interface{}

// HandlerFunc represents an event handler callback
type HandlerFunc func(Event)

// Handler represents an event handler
type Handler struct {
	Func     HandlerFunc // The callback that this Handler refers to
	Priority int         // The priority of this callback, lower is higher
	ID       int         // The ID of this callback
}

// Manager is an event bus. It allows you to hook callbacks onto string based event names, and fire them later. Use
// of Manager objects from multiple goroutines is permitted
type Manager struct {
	events Map
	m      sync.RWMutex
	curId  int
}

func (m *Manager) nextID() int {
	m.m.Lock()
	defer m.m.Unlock()
	m.curId++
	return m.curId
}

// HasEvent returns whether or not the given string exists as an event on this Manager
func (m *Manager) HasEvent(name string) bool {
	m.m.RLock()
	defer m.m.RUnlock()
	_, ok := m.events[name]
	return ok
}

// Attach adds an event and a callback to the Manager, the returned int is an ID for the attached callback, and can
// be used to detach a callback later
func (m *Manager) Attach(name string, f HandlerFunc, priority int) int {
	id := m.nextID()
	m.m.Lock()
	defer m.m.Unlock()
	if m.events == nil {
		m.events = make(Map)
	}
	m.events[name] = append(m.events[name], Handler{f, priority, id})
	sort.Sort(m.events[name])
	return id
}

// AttachOneShot attaches the function provided to the Manager for one hook, after which the handler will be detached
func (m *Manager) AttachOneShot(name string, f HandlerFunc, priority int) int {
	return m.AttachMultiShot(name, f, priority, 1)
}

// AttachMultiShot attaches the given callback for count number of hook dispatches, once the count is reached, the
// callback will be detached
func (m *Manager) AttachMultiShot(name string, f HandlerFunc, priority int, count int) int {
	callCount := 0
	var id int
	wrapped := func(e Event) {
		callCount++
		defer func() {
			if callCount >= count {
				m.Detach(id)
			}
		}()
		f(e)
	}
	id = m.Attach(name, wrapped, priority)
	return id
}

// Detach removes a given ID from the event Manager. If the event is not found, Detach returns False
func (m *Manager) Detach(id int) bool {
	m.m.Lock()
	defer m.m.Unlock()
	var targetName string
	targetIdx := -1 // Dont mutate while iterating--fun things happen
loop:
	for name, hl := range m.events {
		for i, handler := range hl {
			if handler.ID == id {
				targetName = name
				targetIdx = i
				break loop
			}
		}
	}

	if targetIdx != -1 {
		// TODO: these are pointers (because they're functions) which means that doing this this way is a memory leak
		// 		 See the golang SliceTricks wiki for more info
		m.events[targetName] = append(m.events[targetName][:targetIdx], m.events[targetName][targetIdx+1:]...)
		return true
	}

	return false
}

// Dispatch fires an event down the event bus under the name attached to the given event. If the name does not exist,
// it is silently ignored
func (m *Manager) Dispatch(event Event) {
	m.m.RLock()
	toIterate, ok := m.events[event.Name()]
	m.m.RUnlock()
	if !ok {
		return
	}

	for _, h := range toIterate {
		h.Func(event)
	}
}

// WaitForChan returns a channel that will receive exactly one dispatch of the named event. The Event object is sent
// over the channel. The channel has a buffer, to prevent blocking of the event's dispatch
func (m *Manager) WaitForChan(name string) <-chan Event {
	c := make(chan Event, 1)
	m.AttachOneShot(name, func(event Event) {
		c <- event
		close(c)
	}, PriNorm)
	return c
}

// WaitFor is like WaitForChan but instead of returning a channel, it blocks until the event is fired, and will return
// the Event object used to fire the event
func (m *Manager) WaitFor(name string) Event {
	return <-m.WaitForChan(name)
}
