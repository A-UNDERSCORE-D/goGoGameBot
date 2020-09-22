package irc

import "awesome-dragon.science/go/goGoGameBot/pkg/event"

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

	// TODO: add a goroutine here that tries to regain our original nick if its in use
	i.runtimeNick = newNick
}

func (i *IRC) onNick(source, newNick string) {
	oldNick := i.HumanReadableSource(source)
	if oldNick != i.runtimeNick {
		return
	}

	i.runtimeNick = newNick
}
