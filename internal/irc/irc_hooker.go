package irc

import (
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/irc/ctcp"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// HookMessage hooks on messages to a channel
func (i *IRC) HookMessage(f func(source, channel, message string, isAction bool)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		messageEvent := e.(*MessageEvent)
		if e.IsCancelled() || messageEvent.IsNotice || !strings.HasPrefix(messageEvent.Channel, "#") {
			return
		}
		act := false
		msg := messageEvent.Message
		if out, err := ctcp.Parse(messageEvent.Message); err == nil {
			if out.Command != "ACTION" {
				return
			}
			msg = out.Arg
			act = true
		}
		f(util.UserHost2Canonical(messageEvent.Source), messageEvent.Channel, ircTransformer.MakeIntermediate(msg), act)
	}, event.PriNorm)
}

// HookPrivateMessage hooks on messages to us directly
func (i *IRC) HookPrivateMessage(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("MSG", func(e event.Event) {
		msg := e.(*MessageEvent)
		if e.IsCancelled() || msg.IsNotice || strings.HasPrefix(msg.Channel, "#") {
			return
		}
		f(util.UserHost2Canonical(msg.Source), msg.Channel, ircTransformer.MakeIntermediate(msg.Message))
	}, event.PriNorm)
}

// HookJoin hooks on users joining a channel
func (i *IRC) HookJoin(f func(source, channel string)) {
	i.ParsedEvents.Attach("JOIN", func(e event.Event) {
		join := e.(*JoinEvent)
		f(util.UserHost2Canonical(join.Source), join.Channel)
	}, event.PriNorm)
}

// HookPart hooks on users leaving a channel
func (i *IRC) HookPart(f func(source, channel, message string)) {
	i.ParsedEvents.Attach("PART", func(e event.Event) {
		part := e.(*PartEvent)
		f(util.UserHost2Canonical(part.Source), part.Channel, ircTransformer.MakeIntermediate(part.Message))
	}, event.PriNorm)
}

// HookQuit hooks on users disconnecting
func (i *IRC) HookQuit(f func(source, message string)) {
	i.ParsedEvents.Attach("QUIT", func(e event.Event) {
		quit := e.(*QuitEvent)
		f(util.UserHost2Canonical(quit.Source), ircTransformer.MakeIntermediate(quit.Message))
	}, event.PriNorm)
}

// HookKick hooks on a user being kicked from a channel
func (i *IRC) HookKick(f func(source, channel, target, message string)) {
	i.ParsedEvents.Attach("KICK", func(e event.Event) {
		kick := e.(*KickEvent)
		f(util.UserHost2Canonical(kick.Source), kick.Channel, kick.KickedNick, ircTransformer.MakeIntermediate(kick.Message))
	}, event.PriNorm)
}

// HookNick hoops on a user changing their nickname
func (i *IRC) HookNick(f func(source, newNick string)) {
	i.ParsedEvents.Attach("NICK", func(e event.Event) {
		nick := e.(*NickEvent)
		f(util.UserHost2Canonical(nick.Source), nick.NewNick)
	}, event.PriNorm)
}
