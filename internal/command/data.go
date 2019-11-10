package command

import (
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
)

// Data represents all the data available for a command call
type Data struct {
	FromTerminal bool
	Args         []string
	OriginalArgs string
	Source       string
	Target       string
	Manager      *Manager
	util         DataUtil
}

// DataUtil provides methods for Data to use when returning messages or checking admin levels
type DataUtil interface {
	interfaces.AdminLeveler
	interfaces.Messager
}

const notAllowed = "You are not permitted to use this command"

// CheckPerms verifies that the admin level of the source user is at or above the requiredLevel
func (d *Data) CheckPerms(requiredLevel int) bool {
	if d.FromTerminal || d.util.AdminLevel(d.Source) >= requiredLevel {
		return true
	}
	d.ReturnNotice(notAllowed)
	return false
}

// SendNotice sends an IRC notice to the given target with the given message
func (d *Data) SendNotice(target, msg string) { d.util.SendNotice(target, msg) }

// SendTargetNotice is a shortcut to SendNotice that sets the target of the notice to the target of the Data object
func (d *Data) SendTargetNotice(msg string) { d.SendNotice(d.Target, msg) }

// SendSourceNotice is a shortcut to SendNotice that sets the target of the notice to the nick of the source of the data
// object
func (d *Data) SendSourceNotice(msg string) { d.SendNotice(d.Source, msg) }

// SendMessage sends an IRC privmsg to the given target
func (d *Data) SendMessage(target, msg string) { d.util.SendMessage(target, msg) }

// SendTargetMessage is a shortcut to SendMessage that sets the target for the privmsg to the target of the Data object
func (d *Data) SendTargetMessage(msg string) { d.SendMessage(d.Target, msg) }

// SendSourceMessage is a shortcut to SendMessage that sets the target for the privmsg to the nick of the source on the
// data object
func (d *Data) SendSourceMessage(msg string) { d.SendMessage(d.Source, msg) }

// ReturnNotice either sends a notice to the caller of a command, or logs the notice to the INFO level using the Manager
// on the Data object. The decision on where to send the message is based on whether or not the source of the command
// that the Data represents is IRC or not
func (d *Data) ReturnNotice(msg string) {
	if d.FromTerminal {
		d.Manager.Logger.Info(msg)
	} else {
		d.SendSourceNotice(msg)
	}
}

// ReturnMessage either sends a notice to the caller of a command, or logs the message to the INFO level using the
// Manager on the Data object. The decision on where to send the message is based on whether or not the source of
// the command that the Data represents is IRC or not
func (d *Data) ReturnMessage(msg string) {
	if d.FromTerminal {
		d.Manager.Logger.Info(msg)
	} else {
		d.SendTargetMessage(msg)
	}
}

// String implements the stringer interface
func (d *Data) String() string { return strings.Join(d.Args, " ") }
