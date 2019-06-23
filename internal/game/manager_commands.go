package game

import (
	"fmt"
	"strings"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
)

const (
	gameNotExist       = "game with name %q does not exist"
	gameAlreadyRunning = "game %q is already running (did you mean restart?)"
	gameNotRunning     = "game %q is not running"
)

func checkArgs(args []string, minLen int, msg string, resp interfaces.CommandResponder) bool {
	if len(args) >= minLen {
		return true
	}

	resp.ReturnNotice(msg)
	return false
}

func (m *Manager) startGameCmd(_ bool, args []string, _ ircutils.UserHost, _ string, responder interfaces.CommandResponder) {
	if !checkArgs(args, 1, "start requires at least one argument", responder) {
		return
	}
	for _, name := range args {
		g := m.GetGameFromName(name)
		if g == nil {
			responder.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}

		if g.IsRunning() {
			responder.ReturnNotice(fmt.Sprintf(gameAlreadyRunning, name))
			continue
		}

		go g.Run()
	}
}

func (m *Manager) stopGameCmd(_ bool, args []string, _ ircutils.UserHost, _ string, responder interfaces.CommandResponder) {
	if !checkArgs(args, 1, "stop requires at least one argument", responder) {
		return
	}

	for _, name := range args {
		g := m.GetGameFromName(name)
		if g == nil {
			responder.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}

		if !g.IsRunning() {
			responder.ReturnNotice(fmt.Sprintf(gameNotRunning, name))
			continue
		}

		go g.StopOrKill()
	}
}

func (m *Manager) rawGameCmd(_ bool, args []string, _ ircutils.UserHost, _ string, responder interfaces.CommandResponder) {
	if !checkArgs(args, 2, "raw requires a game to target, and the message to send", responder) {
		return
	}

	name := args[0]
	msg := strings.Join(args[1:], " ")
	g := m.GetGameFromName(name)
	if g == nil {
		responder.ReturnNotice(fmt.Sprintf(gameNotExist, name))
		return
	}

	if !g.IsRunning() {
		responder.ReturnNotice(fmt.Sprintf(gameNotRunning, name))
		return
	}

	if _, err := g.WriteString(msg); err != nil {
		responder.ReturnNotice(fmt.Sprintf("an error occurred while writing to game %q", name))
		m.Warnf("could not write message %q to game %q: %s", msg, name, err)
	}
}

func (m *Manager) restartGameCmd(_ bool, args []string, _ ircutils.UserHost, _ string, responder interfaces.CommandResponder) {
	if !checkArgs(args, 1, "restart at least one game to restart", responder) {
		return
	}

	for _, name := range args {
		g := m.GetGameFromName(name)
		if g == nil {
			responder.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}
		go restartGame(g, responder) // Restart games in parallel, while still waiting for each to stop on their own
	}
}

func restartGame(game interfaces.Game, responder interfaces.CommandResponder) {
	if !game.IsRunning() {
		responder.ReturnNotice(fmt.Sprintf(gameNotRunning, game.GetName()))
		return
	}
	if err := game.StopOrKill(); err != nil {
		responder.ReturnNotice(fmt.Sprintf("error occurred while restarting game %q: %s", game.GetName(), err))
		return
	}
	go game.Run()
}
