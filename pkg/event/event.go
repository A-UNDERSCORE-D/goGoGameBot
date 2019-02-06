package event

import (
    "sort"
    "sync"
    "time"
)

// This is a reimplementation of eventmgr found at github.com/goshuirc/eventmgr with an ID system added. The original idea is theirs.

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

type Map map[string]HandlerList
type ArgMap map[string]interface{}

type HandlerFunc func(string, ArgMap)
type Handler struct {
    Func     HandlerFunc
    Priority int
    Id       int64
}

type Manager struct {
    events Map
    mutex  sync.Mutex
}

func (m *Manager) HasEvent(name string) bool {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    _, ok := m.events[name]
    return ok
}

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

func (m *Manager) Detach(id int64) bool {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    var targetName string
    targetIdx := -1 // Dont mutate while iterating--fun things happen
    loop:
    for name, hl := range m.events {
        for i, handler := range hl {
            if handler.Id == id {
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
