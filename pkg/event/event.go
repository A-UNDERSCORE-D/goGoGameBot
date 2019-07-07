package event

import (
	"sort"
	"sync"
	"time"
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
type HandlerFunc func(string, ArgMap)

// Handler represents an event handler
type Handler struct {
	Func     HandlerFunc // The callback that this Handler refers to
	Priority int         // The priority of this callback, lower is higher
	ID       int64       // The ID of this callback
}

// Manager is an event system. It allows you to hook callbacks onto string based event names, and fire them later. Use
// of Manager objects from multiple goroutines is permitted
type Manager struct {
	events Map
	mutex  sync.Mutex
}

// HasEvent returns whether or not the given string exists as an event on this Manager
func (m *Manager) HasEvent(name string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.events[name]
	return ok
}

// Attach adds an event and a callback to the Manager, the returned int64 is an ID for the attached callback, and can
// be used to detach a callback later
func (m *Manager) Attach(name string, f HandlerFunc, priority int) int64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.events == nil {
		m.events = make(Map)
	}
	id := time.Now().UnixNano()
	m.events[name] = append(m.events[name], Handler{f, priority, id})
	sort.Sort(m.events[name])
	return id
}

// Detach removes a given ID from the event Manager. If the event is not found, Detach returns False
func (m *Manager) Detach(id int64) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
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
		m.events[targetName] = append(m.events[targetName][:targetIdx], m.events[targetName][targetIdx+1:]...)
		return true
	}

	return false
}

// Dispatch fires the event corresponding to the given string with the given ArgMap. If the event doesnt exist on this
// Manager, Dispatch silently ignores the call
func (m *Manager) Dispatch(name string, argMap ArgMap) {
	m.mutex.Lock()
	toIterate, ok := m.events[name]
	m.mutex.Unlock()
	if !ok {
		return
	}

	for _, h := range toIterate {
		h.Func(name, argMap)
	}
}
