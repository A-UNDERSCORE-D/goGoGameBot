package irc

import (
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

func (i *IRC) HookMessage(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		msg := e.(*MessageEvent)
		if e.IsCancelled() || msg.IsNotice || !strings.HasPrefix(msg.Channel, "#") {
			return
		}
		f(util.UserHost2Canonical(msg.Source), msg.Channel, msg.Message)

	}, event.PriNorm)
}

func (i *IRC) HookPrivateMessage(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		msg := e.(*MessageEvent)
		if e.IsCancelled() || msg.IsNotice || strings.HasPrefix(msg.Channel, "#") {
			return
		}
		f(util.UserHost2Canonical(msg.Source), msg.Channel, msg.Message)

	}, event.PriNorm)
}

func (i *IRC) HookJoin(f func(source, channel string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		join := e.(*JoinEvent)
		f(util.UserHost2Canonical(join.Source), join.Channel)
	}, event.PriNorm)
}

func (i *IRC) HookPart(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("PART", func(e event.Event) {
		part := e.(*PartEvent)
		f(util.UserHost2Canonical(part.Source), part.Channel, part.Message)
	}, event.PriNorm)
}

func (i *IRC) HookQuit(f func(source, message string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		quit := e.(*QuitEvent)
		f(util.UserHost2Canonical(quit.Source), quit.Message)
	}, event.PriNorm)
}

func (i *IRC) HookKick(f func(source, channel, target, message string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		kick := e.(*KickEvent)
		f(util.UserHost2Canonical(kick.Source), kick.Channel, kick.KickedNick, kick.Message)
	}, event.PriNorm)
}

func (i *IRC) HookNick(f func(source, newNick string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		nick := e.(*NickEvent)
		f(util.UserHost2Canonical(nick.Source), nick.NewNick)
	}, event.PriNorm)
}
