package command

import (
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/goshuirc/irc-go/ircutils"
)

// Data represents all the data available for a command call
type Data struct {
	IsFromIRC    bool
	Args         []string
	OriginalArgs string
	Source       ircutils.UserHost
	Target       string
	Manager      *Manager
}

// CheckPerms verifies that the admin level of the source user is at or above the requiredLevel
func (d *Data) CheckPerms(requiredLevel int) bool {
	return d.Manager.CheckAdmin(d, requiredLevel)
}

// SendNotice sends an IRC notice to the given target with the given message
func (d *Data) SendNotice(target, msg string) {
	d.Manager.messenger.SendNotice(target, ircfmt.Unescape(msg))
}

// SendTargetNotice is a shortcut to SendNotice that sets the target of the notice to the target of the Data object
func (d *Data) SendTargetNotice(msg string) {
	d.SendNotice(d.Target, msg)
}

// SendSourceNotice is a shortcut to SendNotice that sets the target of the notice to the nick of the source of the data
// object
func (d *Data) SendSourceNotice(msg string) {
	d.SendNotice(d.Source.Nick, msg)
}

// SendPrivmsg sends an IRC privmsg to the given target
func (d *Data) SendPrivmsg(target, msg string) {
	d.Manager.messenger.SendPrivmsg(target, ircfmt.Unescape(msg))
}

// SendTargetMessage is a shortcut to SendPrivmsg that sets the target for the privmsg to the target of the Data object
func (d *Data) SendTargetMessage(msg string) {
	d.SendPrivmsg(d.Target, msg)
}

// SendSourceMessage is a shortcut to SendPrivmsg that sets the target for the privmsg to the nick of the source on the
// data object
func (d *Data) SendSourceMessage(msg string) {
	d.SendPrivmsg(d.Source.Nick, msg)
}

// SendRawMessage instructs the bot on the Manager to send a raw IRC line
func (d *Data) SendRawMessage(line string) error {
	return d.Manager.messenger.WriteString(line)
}

// ReturnNotice either sends a notice to the caller of a command, or logs the notice to the INFO level using the Manager
// on the Data object. The decision on where to send the message is based on whether or not the source of the command
// that the Data represents is IRC or not
func (d *Data) ReturnNotice(msg string) {
	if d.IsFromIRC {
		d.SendSourceNotice(msg)
	} else {
		d.Manager.Logger.Info(msg)
	}
}

// ReturnMessage either sends a notice to the caller of a command, or logs the message to the INFO level using the
// Manager on the Data object. The decision on where to send the message is based on whether or not the source of
// the command that the Data represents is IRC or not
func (d *Data) ReturnMessage(msg string) {
	if d.IsFromIRC {
		d.SendTargetMessage(msg)
	} else {
		d.Manager.Logger.Info(msg)
	}
}

// String implements the stringer interface
func (d *Data) String() string {
	return strings.Join(d.Args, " ")
}

// SourceMask returns the mask of the source in the canonical nickname!username@host format
func (d *Data) SourceMask() string {
	out := strings.Builder{}
	out.WriteString(d.Source.Nick)
	if d.Source.User != "" {
		out.WriteRune('!')
		out.WriteString(d.Source.User)
	}
	if d.Source.Host != "" {
		out.WriteRune('@')
		out.WriteString(d.Source.Host)
	}
	return out.String()
}
