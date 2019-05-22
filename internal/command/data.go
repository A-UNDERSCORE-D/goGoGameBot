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

func (d *Data) SendTargetNotice(msg string) {
	d.Manager.messenger.SendNotice(d.Target, msg)
}

func (d *Data) SendTargetMessage(msg string) {
	d.Manager.messenger.SendPrivmsg(d.Target, msg)
}

func (d *Data) SendSourceNotice(msg string) {
	d.Manager.messenger.SendNotice(d.Source.Nick, msg)
}

func (d *Data) SendSourceMessage(msg string) {
	d.Manager.messenger.SendPrivmsg(d.Source.Nick, msg)
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
