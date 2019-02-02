package bot

import (
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
    "github.com/goshuirc/irc-go/ircmsg"
    "github.com/goshuirc/irc-go/ircutils"
    "strings"
)

type HandleFunc func(data *CommandData) error

// CommandHandler wraps the event manager on a Bot and uses it to build a command interface
type CommandHandler struct {
    bot    *Bot
    prefix string
    commands map[string][]int64
}

// NewCommandHandler creates a CommandHandler and attaches its primary listener to the event manager on the given bot
func NewCommandHandler(b *Bot, prefixes string) *CommandHandler {
    h := &CommandHandler{bot: b, prefix: prefixes, commands:make(map[string][]int64)}
    b.HookRaw("PRIVMSG", h.mainListener, PriNorm)
    //b.EventMgr.Attach("RAW_PRIVMSG", h.mainListener, PriNorm)
    return h
}

// mainListener is the main PRIVMSG handler for the command handler. It dispatches events for commands after they have
// been broken up into a CommandData
func (h *CommandHandler) mainListener(line ircmsg.IrcMessage, b *Bot) {
    splitMsg := util.CleanSplitOnSpace(line.Params[1])
    if len(splitMsg) < 1 {
        return // There's nothing there
    }

    var cmd string
    var args []string

    firstWord := strings.ToUpper(splitMsg[0])
    if strings.HasPrefix(firstWord, h.prefix) {
        cmd = firstWord[len(h.prefix):]
        args = splitMsg[1:]
    } else if strings.HasPrefix(firstWord, strings.ToUpper(b.IrcConf.Nick)) && len(firstWord)-len(b.IrcConf.Nick) < 2 && len(splitMsg) > 1 {
        cmd = strings.ToUpper(splitMsg[1])
        args = splitMsg[2:]
    } else {
        return
    }

    target := line.Params[0]
    if target == strings.ToUpper(b.IrcConf.Nick) {
        target = ircutils.ParseUserhost(line.Prefix).Nick
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
    h.internalFireCommand(strings.ToUpper(data.Command), event.ArgMap{"data": data})
}

// internalFireCommand fires the event to run a command if it exists, otherwise it fires the command not found event
func (h *CommandHandler) internalFireCommand(cmd string, im event.ArgMap) {

    go h.bot.EventMgr.Dispatch("CMD", im)

    if h.bot.EventMgr.HasEvent("CMD_"+cmd) {
        go h.bot.EventMgr.Dispatch("CMD_"+cmd, im)
    } else {
        go h.bot.EventMgr.Dispatch("CMDNOTFOUND", im)
    }
}

// RegisterCommand registers a callback with a command
func (h *CommandHandler) RegisterCommand(cmd string, f HandleFunc, priority int, requiresAdmin bool) {
    wrapped := func(event string, infoMap event.ArgMap) {
        data := infoMap["data"].(*CommandData)
        if data.SourceIsIgnored() || data.isCancelled {
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

    id := h.bot.EventMgr.Attach(hookName, wrapped, priority)
    h.commands[hookName] = append(h.commands[hookName], id)
}

func (h *CommandHandler) UnregisterCommand(name string) {
    IDs, ok := h.commands[strings.ToUpper(name)]
    if !ok {
        h.bot.Log.Warnf("Attempt to remove nonexistant command %q", name)
        return
    }
    h.bot.Log.Infof("unregistering command %q", name)
    for _, id := range IDs {
        h.bot.EventMgr.Detach(id)
    }

    delete(h.commands, name)
}

// checkPermissions does runs a glob based permission check on the incoming command, cancelling the command being fired
// if the check fails
func checkPermissions(data *CommandData) error {
    if !data.IsFromIRC {
        return nil
    }

    ok := false
    for _, perm := range data.Bot.Config.Permissions {
        matcher := util.GlobToRegexp(perm.Mask)
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
