package event

import (
    "fmt"
    "testing"
)

var manager Manager

func TestStuff(t *testing.T) {
    manager.Attach("test", func(s string, maps ArgMap) {
        fmt.Printf("1: I was called on %q with argmap %#v\n", s, maps)
    }, 1)
    manager.Attach("test", func(s string, maps ArgMap) {
        fmt.Printf("5: I was called on %q with argmap %#v\n", s, maps)
    }, 5)
    manager.Attach("test", func(s string, maps ArgMap) {
        fmt.Printf("2: I was called on %q with argmap %#v\n", s, maps)
    }, 2)

    id := manager.Attach("test", func(s string, maps ArgMap) {
        fmt.Printf("3: I was called on %q with argmap %#v\n", s, maps)
    }, 3)

    manager.Dispatch("test", ArgMap{"test": 0})
    manager.Detach(id)
    fmt.Println(manager.events)
    fmt.Println()
    manager.Dispatch("test", ArgMap{"test": 1})
}
