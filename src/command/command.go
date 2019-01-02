package command

import (
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/irc/bot"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "strings"
)

type HandleFunc func(data Data) error

// Handler wraps the event manager on a Bot and uses it to build a command interface
type Handler struct {
    bot    *bot.Bot
    prefix string
}

// NewHandler creates a Handler and attaches its primary listener to the event manager on the given bot
func NewHandler(b *bot.Bot, prefixes string) *Handler {
    h := &Handler{bot: b, prefix: prefixes}
    b.EventMgr.Attach("RAW_PRIVMSG", h.mainListener, bot.PriNorm)
    return h
}

// mainListener is the main PRIVMSG handler for the command handler. It dispatches events for commands after they have
// been broken up into a Data
func (h *Handler) mainListener(event string, infoMap eventmgr.InfoMap) {
    line := infoMap["line"].(ircmsg.IrcMessage)
    msg := line.Params[1]
    if len(msg) < 1 {
        return
    }

    target := line.Params[0]
    sMsg := strings.Split(msg, " ")
    cmd := strings.ToUpper(sMsg[0])

    if !strings.HasPrefix(h.prefix, cmd) {
        // Not a command we understand
        return
    }

    var args []string
    if len(sMsg) > 1 {
        args = sMsg[1:]
    } else {
        args = []string{}
    }

    im := eventmgr.NewInfoMap()

    im["data"] = Data{
        Command:   cmd,
        Target:    target,
        Args:      args,
        Line:      &line,
        Source:    line.Prefix,
        IsFromIRC: true,
    }

    h.bot.EventMgr.Dispatch("CMD_"+cmd, im)

}

// FireCommand fires the event to run a command if it exists, otherwise it fires the command not found event
func (h *Handler) FireCommand(cmd string, im eventmgr.InfoMap) {

    go h.bot.EventMgr.Dispatch("CMD", im)

    if _, exists := h.bot.EventMgr.Events["CMD_"+cmd]; exists {
        go h.bot.EventMgr.Dispatch("CMD_"+cmd, im)
    } else {
        go h.bot.EventMgr.Dispatch("CMDNOTFOUND", im)
    }
}

// RegisterCommand registers a callback with a command
func (h *Handler) RegisterCommand(cmd string, f HandleFunc, priority int) {
    wrapped := func(event string, infoMap eventmgr.InfoMap){
        data := infoMap["data"].(Data)

        if data.IsCancelled() {
            return
        }

        if err := f(data); err != nil {
            infoMap["error"] = err
            go h.bot.EventMgr.Dispatch("ERR", infoMap)
        }
    }

    h.bot.EventMgr.Attach("CMD_" + strings.ToUpper(cmd), wrapped, priority)
}
