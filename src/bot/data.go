package bot

import (
    "github.com/goshuirc/irc-go/ircmsg"
    "github.com/goshuirc/irc-go/ircutils"
    "github.com/pkg/errors"
    "strings"
    "sync"
)

// CommandData holds all the data for a command currently being fired
type CommandData struct {
    Command     string
    Target      string
    Args        []string
    Line        *ircmsg.IrcMessage
    Source      string
    IsFromIRC   bool
    cancelMutex sync.Mutex
    isCancelled bool
    Bot         *Bot
}

func (d *CommandData) ArgCount() int {
    return len(d.Args)
}

// CommandData.ArgEol returns a list of the
func (d *CommandData) ArgEol() []string {
    var out []string
    for i, _ := range d.Args {
        out = append(out, strings.Join(d.Args[:i], " "))
    }

    return out
}

func (d *CommandData) UserHost() (ircutils.UserHost, error) {
    if d.IsFromIRC {
        return ircutils.ParseUserhost(d.Source), nil
    }

    return ircutils.UserHost{}, errors.New("cannot parse a non-bot source to a UserHost")
}

func (d *CommandData) SetCancelled(toSet bool) {
    d.cancelMutex.Lock()
    defer d.cancelMutex.Unlock()
    d.isCancelled = toSet
}

func (d *CommandData) IsCancelled() bool {
    d.cancelMutex.Lock()
    defer d.cancelMutex.Unlock()
    return d.isCancelled
}

func (d *CommandData) ArgString() string {
    return strings.Join(d.Args, " ")
}
