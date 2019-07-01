package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
)

func (g *Game) checkError(err error) {
	if err != nil {
		g.manager.Error(err)
	}
}

// IsRunning returns whether or not the process is currently running
func (g *Game) IsRunning() bool {
	return g.process.IsRunning()
}

// MapColours maps any IRC colours found in the string to the colour map on the game
func (g *Game) MapColours(s string) string {
	if g.chatBridge.colourMap == nil {
		g.Warn("Colour map is nil. returning stripped string instead")
		return ircfmt.Strip(s)
	}
	return g.chatBridge.colourMap.Replace(ircfmt.Escape(s))
}

func (g *Game) prefixMsg(args ...interface{}) string {
	return fmt.Sprintf("[%s] %s", g.name, fmt.Sprint(args...))
}

func (g *Game) sendToMsgChan(args ...interface{}) {
	g.manager.bot.SendPrivmsg(g.controlChannels.msg, g.prefixMsg(args...))
}

func (g *Game) sendToAdminChan(args ...interface{}) {
	g.manager.bot.SendPrivmsg(g.controlChannels.admin, g.prefixMsg(args...))
}

func (g *Game) writeToAllOthers(msg string) {
	msg = strings.ReplaceAll(msg, "\u200b", "")
	g.manager.ForEachGame(func(game interfaces.Game) {
		if !g.IsRunning() {
			return
		}
		game.SendLineFromOtherGame(msg, g)
	}, []interfaces.Game{g})
}

func (g *Game) templSendToAdminChan(v ...interface{}) string {
	msg := fmt.Sprint(v...)
	g.sendToAdminChan(msg)
	return msg
}

func (g *Game) templSendToMsgChan(v ...interface{}) string {
	msg := fmt.Sprint(v...)
	g.sendToMsgChan(msg)
	return msg
}

func (g *Game) templSendPrivmsg(c string, v ...interface{}) (string, error) {
	if c == "" {
		return "", errors.New("cannot send to a nonexistent target")
	}
	msg := fmt.Sprint(v...)
	g.manager.bot.SendPrivmsg(c, msg)
	return msg, nil
}

func (g *Game) Status() string {
	return g.process.GetStatus()
}
