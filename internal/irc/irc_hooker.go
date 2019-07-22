package irc

import (
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
)

func (i *IRC) HookMessage(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		msg := e.(MessageEvent)
		if e.IsCancelled() || msg.IsNotice || !strings.HasPrefix(msg.Channel, "#") {
			return
		}
		f(msg.Source.Nick, msg.Channel, msg.Message)

	}, event.PriNorm)
}

func (i *IRC) HookPrivateMessage(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		msg := e.(*MessageEvent)
		if e.IsCancelled() || msg.IsNotice || strings.HasPrefix(msg.Channel, "#") {
			return
		}
		f(msg.Source.Nick, msg.Channel, msg.Message)

	}, event.PriNorm)
}

func (i *IRC) HookJoin(f func(source, channel string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		join := e.(*JoinEvent)
		f(join.Source.Nick, join.Channel)
	}, event.PriNorm)
}

func (i *IRC) HookPart(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("PART", func(e event.Event) {
		part := e.(*PartEvent)
		f(part.Source.Nick, part.Channel, part.Message)
	}, event.PriNorm)
}

func (i *IRC) HookQuit(f func(source, message string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		quit := e.(*QuitEvent)
		f(quit.Source.Nick, quit.Message)
	}, event.PriNorm)
}

func (i *IRC) HookKick(f func(source, channel, target, message string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		kick := e.(*KickEvent)
		f(kick.Source.Nick, kick.Channel, kick.KickedNick, kick.Message)
	}, event.PriNorm)
}

func (i *IRC) HookNick(f func(source, newNick string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		nick := e.(*NickEvent)
		f(nick.Source.Nick, nick.NewNick)
	}, event.PriNorm)
}
