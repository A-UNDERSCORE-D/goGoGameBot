package irc

import (
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
)

func event2RawEvent(e event.Event) *RawEvent {
	raw, ok := e.(*RawEvent)
	if !ok {
		return nil
	}
	return raw
}

func (i *IRC) dispatchMessage(e event.Event) {
	raw := event2RawEvent(e)
	if raw == nil || !raw.CommandIs("PRIVMSG", "NOTICE") {
		i.log.Warnf("Got a NOTICE or PRIVMSG message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewMessageEvent("MSG", raw.Line, raw.Time))
}

func (i *IRC) dispatchJoin(e event.Event) {
	var raw *RawEvent
	if raw = event2RawEvent(e); raw == nil || !raw.CommandIs("JOIN") {
		i.log.Warnf("Got a JOIN message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewJoinEvent("JOIN", raw.Line, raw.Time))
}

func (i *IRC) dispatchPart(e event.Event) {
	var raw *RawEvent
	if raw = event2RawEvent(e); raw == nil || !raw.CommandIs("PART") {
		i.log.Warnf("Got a PART message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewPartEvent("PART", raw.Line, raw.Time))
}

func (i *IRC) dispatchQuit(e event.Event) {
	var raw *RawEvent
	if raw = event2RawEvent(e); raw == nil || !raw.CommandIs("QUIT") {
		i.log.Warnf("Got a QUIT message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewQuitEvent("QUIT", raw.Line, raw.Time))
}

func (i *IRC) dispatchKick(e event.Event) {
	var raw *RawEvent
	if raw = event2RawEvent(e); raw == nil || !raw.CommandIs("KICK") {
		i.log.Warnf("Got a KICK message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewKickEvent("KICK", raw.Line, raw.Time))
}

func (i *IRC) dispatchNick(e event.Event) {
	var raw *RawEvent
	if raw = event2RawEvent(e); raw == nil || !raw.CommandIs("NICK") {
		i.log.Warnf("Got a NICK message that was invalid: %v", raw)
		return
	}
	i.ParsedEvents.Dispatch(NewNickEvent("NICK", raw.Line, raw.Time))
}
