package command

import (
	"strings"

	"github.com/goshuirc/irc-go/ircutils"
)

type Data struct {
	IsFromIRC        bool
	Args             []string
	OriginalArgs     string
	Source           ircutils.UserHost
	Target           string
	//SourceAdminLevel int
	Manager          *Manager
}

func (d *Data) CheckPerms(requiredLevel int) bool {
	return d.Manager.CheckAdmin(d, requiredLevel)
}

func (d *Data) SendNotice(target, msg string) {
	d.Manager.messenger.SendNotice(target, msg)
}

func (d *Data) SendTargetNotice(msg string) {
	d.SendNotice(d.Target, msg)
}

func (d *Data) SendSourceNotice(msg string) {
	d.SendNotice(d.Source.Nick, msg)
}

func (d *Data) SendPrivmsg(target, msg string) {
	d.Manager.messenger.SendPrivmsg(target, msg)
}
func (d *Data) SendTargetMessage(msg string) {
	d.SendPrivmsg(d.Target, msg)
}

func (d *Data) SendSourceMessage(msg string) {
	d.SendPrivmsg(d.Source.Nick, msg)
}

func (d *Data) SendRawMessage(line string) error {
	return d.Manager.messenger.WriteString(line)
}

func (d *Data) String() string {
	return strings.Join(d.Args, " ")
}

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
