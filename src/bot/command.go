package bot

import (
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/util"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "regexp"
    "strings"
)

type HandleFunc func(data *CommandData) error

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

    h.FireCommand(&CommandData{
        Command:   cmd,
        Target:    target,
        Args:      args,
        Line:      &line,
        Source:    line.Prefix,
        IsFromIRC: true,
    })
}

// FireCommand preps data for a command to be fired and uses internalFireCommand to fire it
func (h *CommandHandler) FireCommand(data *CommandData) {
    if data.Bot == nil {
        data.Bot = h.bot
    }

    im := eventmgr.NewInfoMap()
    im["data"] = data
    h.internalFireCommand(strings.ToUpper(data.Command), im)
}

// internalFireCommand fires the event to run a command if it exists, otherwise it fires the command not found event
func (h *CommandHandler) internalFireCommand(cmd string, im eventmgr.InfoMap) {

    go h.bot.EventMgr.Dispatch("CMD", im)

    if _, exists := h.bot.EventMgr.Events["CMD_"+cmd]; exists {
        go h.bot.EventMgr.Dispatch("CMD_"+cmd, im)
    } else {
        go h.bot.EventMgr.Dispatch("CMDNOTFOUND", im)
    }
}

// RegisterCommand registers a callback with a command
func (h *CommandHandler) RegisterCommand(cmd string, f HandleFunc, priority int, requiresAdmin bool) {
    wrapped := func(event string, infoMap eventmgr.InfoMap) {
        data := infoMap["data"].(*CommandData)
        if data.IsCancelled() {
            return
        }

        if err := f(data); err != nil {
            infoMap["Error"] = err
            go h.bot.EventMgr.Dispatch("ERR", infoMap)
        }
    }

    hookName := "CMD_" + strings.ToUpper(cmd)
    if requiresAdmin {
        h.RegisterCommand(cmd, checkPermissions, -1, false)
    }

    h.bot.EventMgr.Attach(hookName, wrapped, priority)
}

// checkPermissions does runs a glob based permission check on the incoming command, cancelling the command being fired
// if the check fails
func checkPermissions(data *CommandData) error {
    if !data.IsFromIRC {
        return nil
    }

    ok := false
    for _, perm := range data.Bot.Config.Permissions {
        matcher := regexp.MustCompile(util.GlobToRegexp(perm.Mask)) // TODO: recompiling this every time is dumb. They should be compiled and stored
        if matcher.MatchString(data.Source) {
            ok = true
            break
        }
    }

    if !ok {
        data.SetCancelled(true)
        target, _ := data.UserHost()
        _ = data.Bot.WriteLine(
            util.MakeSimpleIRCLine("NOTICE", target.Nick, "You are not permitted to use this command"),
        )
    }
    return nil
}

// rawCommand is a command handler that allows the user to send raw IRC lines from the bot
func rawCommand(data *CommandData) error {
    if len(data.Args) < 1 {
        if data.IsFromIRC {
            target, _ := data.UserHost()
            _ = data.Bot.WriteLine(
                util.MakeSimpleIRCLine("NOTICE", target.Nick, "cannot have an empty command"),
                )
        } else {
            data.Bot.Log.Warn("Cannot have an empty command")
        }
        return nil
    }

    toSend := strings.TrimRight(data.ArgString(), "\r\n") + "\r\n"
    _, err := data.Bot.writeRaw([]byte(toSend))
    return err
}
