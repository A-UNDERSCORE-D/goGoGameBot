package game

import (
	"fmt"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/command"
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

func (m *Manager) startGameCmd(data *command.Data) {
	if !checkArgs(data.Args, 1, "start requires at least one argument", data) {
		return
	}
	for _, name := range data.Args {
		g := m.GetGameFromName(name)
		if g == nil {
			data.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}

		if g.IsRunning() {
			data.ReturnNotice(fmt.Sprintf(gameAlreadyRunning, name))
			continue
		}

		go g.Run()
	}
}

func (m *Manager) stopGameCmd(data *command.Data) {
	if !checkArgs(data.Args, 1, "stop requires at least one argument", data) {
		return
	}

	for _, name := range data.Args {
		g := m.GetGameFromName(name)
		if g == nil {
			data.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}

		if !g.IsRunning() {
			data.ReturnNotice(fmt.Sprintf(gameNotRunning, name))
			continue
		}

		go g.StopOrKill()
	}
}

func (m *Manager) rawGameCmd(data *command.Data) {
	if !checkArgs(data.Args, 2, "raw requires a game to target, and the message to send", data) {
		return
	}

	name := data.Args[0]
	msg := strings.Join(data.Args[1:], " ")
	g := m.GetGameFromName(name)
	if g == nil {
		data.ReturnNotice(fmt.Sprintf(gameNotExist, name))
		return
	}

	if !g.IsRunning() {
		data.ReturnNotice(fmt.Sprintf(gameNotRunning, name))
		return
	}

	if _, err := g.WriteString(msg); err != nil {
		data.ReturnNotice(fmt.Sprintf("an error occurred while writing to game %q", name))
		m.Warnf("could not write message %q to game %q: %s", msg, name, err)
	}
}

func (m *Manager) restartGameCmd(data *command.Data) {
	if !checkArgs(data.Args, 1, "restart at least one game to restart", data) {
		return
	}

	for _, name := range data.Args {
		g := m.GetGameFromName(name)
		if g == nil {
			data.ReturnNotice(fmt.Sprintf(gameNotExist, name))
			continue
		}
		go restartGame(g, data) // Restart games in parallel, while still waiting for each to stop on their own
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

func (m *Manager) stopCmd(data *command.Data) {
	msg := ""
	if len(data.Args) > 0 {
		msg = strings.Join(data.Args, " ")
	}
	m.Stop(msg, false)
}

func (m *Manager) restartCmd(data *command.Data) {
	msg := ""
	if len(data.Args) > 0 {
		msg = strings.Join(data.Args, " ")
	}
	m.Stop(msg, true)
}
