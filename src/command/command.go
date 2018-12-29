package command

import (
    "fmt"
    "log"
    "strings"
)

func init() {
    Instance = &Handler{commands: make(map[string]HandleFunc), NotFoundHandler: defaultNotFound}
}

// a HandleFunc takes the full command it got, any arguments, and the source of the command. the bool returned is a
// success indicator

var Instance *Handler
type HandleFunc func(cmd string, args []string, source string) bool
type Handler struct {
    // TODO: Maybe it would be nice for multiple handlers to go on one command? if I do that it needs to have some sort
    //       of priority system. maybe turn HandleFunc into a struct with some numbers on it, and store a list here?
    commands        map[string]HandleFunc
    NotFoundHandler HandleFunc
}

func defaultNotFound(cmd string, args []string, source string) bool {
    log.Printf("unknown command: %q", cmd)
    return false
}

func (h *Handler) RegisterCommand(cmd string, f HandleFunc) error {
    if _, ok := h.commands[cmd]; ok {
        return fmt.Errorf("%q already exists as a command", cmd)
    }

    if strings.Contains(" ", cmd) {
        return fmt.Errorf("commands may not contain spaces (%q)", cmd)
    }

    h.commands[cmd] = f
    return nil
}

func (h *Handler) HandleCommand(cmd string, args []string, source string) {
    if hf, ok := h.commands[cmd]; ok {
        hf(cmd, args, source)
    } else {
        h.NotFoundHandler(cmd, args, source)
    }
}
