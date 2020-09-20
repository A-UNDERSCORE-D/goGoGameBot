package game

import (
	"fmt"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/command"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config/tomlconf"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/systemstats"
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

		go func() {
			if err := g.Run(); err != nil {
				m.Logger.Warnf("got an error from game.Run on %s: %s", g.GetName(), err)
			}
		}()
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

		go func(g interfaces.Game) {
			if err := g.StopOrKill(); err != nil {
				m.Warnf("error occurred while stopping game %s: %q", g.GetName(), err)
			}
		}(g)
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

	go func() {
		_ = game.Run()
	}()
}

func (m *Manager) stopCmd(data *command.Data) {
	msg := "Stop requested"
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

func (m *Manager) reloadCmd(data *command.Data) {
	data.ReturnMessage("reloading config")

	newConf, err := tomlconf.GetConfig(m.rootConf.OriginalPath)
	if err != nil {
		m.Error(err)
		data.ReturnMessage("reload failed")

		return
	}

	if err := m.reload(newConf); err != nil {
		m.Error(err)
		data.ReturnMessage("reload failed")

		return
	}

	data.ReturnMessage("reload complete")
}

func (m *Manager) statusCmd(data *command.Data) {
	ourStats := fmt.Sprintf("%s %s", systemstats.GetStats(), m.bot.Status())

	if len(data.Args) == 0 {
		data.ReturnMessage(ourStats)
		return
	}

	for _, v := range data.Args {
		switch {
		case v == "all":
			data.ReturnMessage(ourStats)
			m.ForEachGame(
				func(g interfaces.Game) {
					data.ReturnMessage(fmt.Sprintf("[%s] %s. (%s)", g.GetName(), g.Status(), g.GetComment()))
				}, nil,
			)

			return

		case m.GameExists(v):
			g := m.GetGameFromName(v)
			data.ReturnMessage(fmt.Sprintf("[%s] %s", g.GetName(), g.Status()))
		default:
			data.ReturnNotice(fmt.Sprintf("unknown game %q", v))
		}
	}
}

func (m *Manager) reconnectCmd(data *command.Data) {
	m.reconnecting.Set(true)

	msg := "reconnecting"

	if len(data.Args) > 0 {
		msg = strings.Join(data.Args, " ")
	}

	m.bot.Disconnect(msg)
}

func (m *Manager) rawCmd(data *command.Data) {
	if len(data.Args) == 0 {
		data.ReturnNotice("raw requires an argument")
	}

	m.bot.SendRaw(strings.Join(data.Args, " "))
}
