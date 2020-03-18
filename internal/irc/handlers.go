package irc

import "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"

func (i *IRC) handleNickInUse(e event.Event) {
	rawEvent := event2RawEvent(e)
	if rawEvent == nil {
		i.log.Warn("Got an invalid 433 event")
		return
	}

	newNick := rawEvent.Line.Params[1] + "_"

	if _, err := i.writeLine("NICK", newNick); err != nil {
		i.log.Warnf("Error while updating nick")
	}

	i.Nick = newNick
}

func (i *IRC) onNick(source, newNick string) {
	oldNick := i.HumanReadableSource(source)
	if oldNick != i.Nick {
		return
	}

	i.Nick = newNick
}
