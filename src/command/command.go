package command

import (
    "fmt"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "log"
    "strings"
)

func init() {
    Instance = &Handler{commands: make(map[string]HandleFunc), NotFoundHandler: defaultNotFound}
}

// a HandleFunc takes the full command it got, any arguments, and the source of the command. the bool returned is a
// success indicator
// TODO: Honestly this could deal with a rewrite. I didnt expect to have the event handler that I now have and using it with
//       prefixes heads back to the event based system on older bots like gonzo and some of my own bots. Its a good system
//       that works well.
var Instance *Handler
type HandleFunc func(cmd string, args []string, source string, fromIRC bool) bool
type Handler struct {
    // TODO: Maybe it would be nice for multiple handlers to go on one command? if I do that it needs to have some sort
    //       of priority system. maybe turn HandleFunc into a struct with some numbers on it, and store a list here?
    commands        map[string]HandleFunc
    NotFoundHandler HandleFunc
}

func defaultNotFound(cmd string, args []string, source string, fromIRC bool) bool {
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

func (h *Handler) HandleCommand(cmd string, args []string, source string, fromIRC bool) {
    if hf, ok := h.commands[cmd]; ok {
        go hf(cmd, args, source, fromIRC)
    } else {
        h.NotFoundHandler(cmd, args, source, fromIRC)
    }
}

func (h *Handler) EventListener(event string, info eventmgr.InfoMap) {
    line := info["line"].(ircmsg.IrcMessage)
    msg := line.Params[len(line.Params) -1]
    splitCmd := strings.Split(msg, " ")

    h.HandleCommand(splitCmd[0], splitCmd[1:], line.Prefix[1:], true)
}
