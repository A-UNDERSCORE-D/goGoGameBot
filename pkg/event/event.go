package event

import (
    "sort"
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
    Events Map
}

func (m *Manager) Attach(name string, f HandlerFunc, priority int) int64 {
    if m.Events == nil {
        m.Events = make(Map)
    }
    id := time.Now().UnixNano()
    m.Events[name] = append(m.Events[name], Handler{f, priority, id})
    sort.Sort(m.Events[name])
    return id
}

func (m *Manager) Detach(id int64) bool {
    var targetName string
    targetIdx := -1 // Dont mutate while iterating--fun things happen
    for name, hl := range m.Events {
        for i, handler := range hl {
            if handler.Id == id {
                targetName = name
                targetIdx = i
                break
            }
        }
    }

    if targetIdx != -1 {
       m.Events[targetName] = append(m.Events[targetName][:targetIdx], m.Events[targetName][targetIdx+1:]...)
       return true
    }

    return false
}

func (m *Manager) Dispatch(name string, argMap ArgMap) {
    for _, h := range m.Events[name] {
        h.Func(name, argMap)
    }
}
