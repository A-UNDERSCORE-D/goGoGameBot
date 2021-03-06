package game

import (
	"errors"
	"fmt"
	"strings"

	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
)

func (g *Game) checkError(err error) {
	if err != nil {
		g.manager.Error(err)
	}
}

// IsRunning returns whether or not the transport is currently running
func (g *Game) IsRunning() bool {
	return g.transport.IsRunning()
}

func (g *Game) prefixMsg(args ...interface{}) string {
	return fmt.Sprintf("[%s] %s", g.name, fmt.Sprint(args...))
}

func (g *Game) sendToBridgedChannel(args ...interface{}) {
	g.manager.bot.SendMessage(g.chatBridge.channel, g.prefixMsg(args...))
}

func (g *Game) writeToAllOthers(msg string) {
	msg = strings.ReplaceAll(msg, "\u200b", "")

	g.manager.ForEachGame(func(game interfaces.Game) {
		if !game.IsRunning() {
			return
		}

		game.SendLineFromOtherGame(msg, g)
	}, []interfaces.Game{g})
}

func (g *Game) templSendToMsgChan(v ...interface{}) string {
	msg := fmt.Sprint(v...)
	g.sendToBridgedChannel(msg)

	return msg
}

func (g *Game) templSendMessage(c string, v ...interface{}) (string, error) {
	if c == "" {
		return "", errors.New("cannot send to a nonexistent target")
	}

	msg := fmt.Sprint(v...)
	g.manager.bot.SendMessage(c, msg)

	return msg, nil
}

// Status returns the status of the game's transport as a string
func (g *Game) Status() string {
	return g.transport.GetHumanStatus()
}
