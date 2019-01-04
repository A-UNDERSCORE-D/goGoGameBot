package bot

import (
    "fmt"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "strings"
)

type HandleFunc func(data CommandData) error

// CommandHandler wraps the event manager on a Bot and uses it to build a command interface
type CommandHandler struct {
    bot    *Bot
    prefix string
}

// NewCommandHandler creates a CommandHandler and attaches its primary listener to the event manager on the given bot
func NewCommandHandler(b *Bot, prefixes string) *CommandHandler {
    h := &CommandHandler{bot: b, prefix: prefixes}
    b.EventMgr.Attach("RAW_PRIVMSG", h.mainListener, PriNorm)
    return h
}

// mainListener is the main PRIVMSG handler for the command handler. It dispatches events for commands after they have
// been broken up into a CommandData
func (h *CommandHandler) mainListener(event string, infoMap eventmgr.InfoMap) {
    line := infoMap["line"].(ircmsg.IrcMessage)
    msg := line.Params[1]
    if len(msg) < 1 {
        return
    }

    target := line.Params[0]
    sMsg := strings.Split(msg, " ")
    cmd := strings.ToUpper(sMsg[0])

    if !strings.HasPrefix(cmd, h.prefix) {
        // Not a command we understand
        return
    }

    cmd = cmd[1:]

    var args []string
    if len(sMsg) > 1 {
        args = sMsg[1:]
    } else {
        args = []string{}
    }

    im := eventmgr.NewInfoMap()

    im["data"] = CommandData{
        Command:   cmd,
        Target:    target,
        Args:      args,
        Line:      &line,
        Source:    line.Prefix,
        IsFromIRC: true,
    }

    h.FireCommand(cmd, im)
}

// FireCommand fires the event to run a command if it exists, otherwise it fires the command not found event
func (h *CommandHandler) FireCommand(cmd string, im eventmgr.InfoMap) {

    go h.bot.EventMgr.Dispatch("CMD", im)

    if _, exists := h.bot.EventMgr.Events["CMD_"+cmd]; exists {
        go h.bot.EventMgr.Dispatch("CMD_"+cmd, im)
    } else {
        go h.bot.EventMgr.Dispatch("CMDNOTFOUND", im)
    }
}

// RegisterCommand registers a callback with a command
func (h *CommandHandler) RegisterCommand(cmd string, f HandleFunc, priority int) {
    wrapped := func(event string, infoMap eventmgr.InfoMap){
        data := infoMap["data"].(CommandData)
        fmt.Println("DATA")
        if data.IsCancelled() {
            return
        }

        if err := f(data); err != nil {
            infoMap["Error"] = err
            go h.bot.EventMgr.Dispatch("ERR", infoMap)
        }
    }

    h.bot.EventMgr.Attach("CMD_" + strings.ToUpper(cmd), wrapped, priority)
}
