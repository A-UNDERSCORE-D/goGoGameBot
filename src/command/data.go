package command

import (
    "github.com/goshuirc/irc-go/ircmsg"
    "github.com/goshuirc/irc-go/ircutils"
    "github.com/pkg/errors"
    "strings"
)

// Data holds all the data for a command currently being fired
type Data struct {
    Command   string
    Target    string
    Args      []string
    Line      *ircmsg.IrcMessage
    Source    string
    IsFromIRC bool
}

func (d *Data) ArgCount() int {
    return len(d.Args)
}

// Data.ArgEol returns a list of the
func (d *Data) ArgEol() []string {
    var out []string
    for i, _ := range d.Args {
        out = append(out, strings.Join(d.Args[:i], " "))
    }

    return out
}

func (d *Data) UserHost() (ircutils.UserHost, error) {
    if d.IsFromIRC {
        return ircutils.ParseUserhost(d.Source), nil
    }

    return ircutils.UserHost{}, errors.New("cannot parse a non-irc source to a UserHost")
}
