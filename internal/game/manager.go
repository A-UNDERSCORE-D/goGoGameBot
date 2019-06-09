package game

import (
	"fmt"
	"sync"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

func NewManager(conf config.GameManager, bot interfaces.Bot, logger *log.Logger) (*Manager, error) {
	m := &Manager{
		bot:    bot,
		Logger: logger.Clone().SetPrefix("GM"),
	}

	m.bot.HookPrivmsg(func(source, target, message string, originalLine ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) { game.OnPrivmsg(source, target, message) }, nil)
	})
	m.bot.HookRaw("JOIN", func(message ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) { game.OnJoin(message.Prefix, message.Params[0]) }, nil)
	}, event.PriNorm)
	m.bot.HookRaw("PART", func(message ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) { game.OnPart(message.Prefix, message.Params[0], message.Params[1]) }, nil)
	}, event.PriNorm)
	m.bot.HookRaw("QUIT", func(message ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) { game.OnQuit(message.Prefix, message.Params[0]) }, nil)
	}, event.PriNorm)
	m.bot.HookRaw("KICK", func(message ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) {
			game.OnKick(message.Prefix, message.Params[0], message.Params[1], message.Params[2])
		}, nil)
	}, event.PriNorm)
	m.bot.HookRaw("NICK", func(message ircmsg.IrcMessage, bot interfaces.Bot) {
		m.ForEachGame(func(game interfaces.Game) { game.OnNick(message.Prefix, message.Params[0]) }, nil)
	}, event.PriNorm)

	var games []interfaces.Game
	for _, g := range conf.Games {
		ng, err := NewGame(g, m)
		if err != nil {
			return nil, fmt.Errorf("could not create game %s: %s", g.Name, err)
		}
		games = append(games, ng)
	}
	m.games = games

	return m, nil
}

type Manager struct {
	games      []interfaces.Game
	gamesMutex sync.RWMutex
	bot        interfaces.Bot
	status     status
	stripMasks []string
	*log.Logger
}

func (m *Manager) String() string {
	m.gamesMutex.RLock()
	defer m.gamesMutex.RUnlock()
	return fmt.Sprintf("game.Manager at %p with %d games attached", m, len(m.games))
}

// ReloadGames uses the passed config values to reload the games stored on it. Any new games
// found in the config are added, rather than reloaded
func (m *Manager) ReloadGames(configs []config.Game) {
	// No need to hold the games mutex as of yet as we're not iterating the games list itself
	for _, conf := range configs {
		switch i := m.gameIdxFromName(conf.Name); i {
		case -1: // Game does not exist
			g, err := NewGame(conf, m)
			if err != nil {
				m.Error(err)
				continue
			}
			if err := m.AddGame(g); err != nil {
				m.Error(err)
				continue
			}
		default:
			m.gamesMutex.RLock()
			g := m.games[i]
			if err := g.UpdateFromConfig(conf); err != nil {
				m.Error(fmt.Errorf("reloading game %s errored: %s", g, err))
			}
			m.gamesMutex.RUnlock()
		}
	}
}

func (m *Manager) gameIdxFromName(name string) int {
	m.gamesMutex.RLock()
	defer m.gamesMutex.RUnlock()
	for i, g := range m.games {
		if g.GetName() == name {
			return i
		}
	}
	return -1
}

// GetGameFromName returns the game represented by the given name. If the game does not exist,
// it returns nil
func (m *Manager) GetGameFromName(name string) interfaces.Game {
	if i := m.gameIdxFromName(name); i != -1 {
		return m.games[i]
	}
	return nil
}

// GameExists returns whether or not the given name exists on any game found on this Manager
func (m *Manager) GameExists(name string) bool {
	m.gamesMutex.RLocker()
	defer m.gamesMutex.RUnlock()
	return m.gameIdxFromName(name) != -1
}

// AddGame adds the given game to the Manager's managed games. If the game already exists, AddGame returns an error
func (m *Manager) AddGame(g interfaces.Game) error {
	if m.GameExists(g.GetName()) {
		return fmt.Errorf("cannot add game %s to manager %v as it already exists", g.GetName(), m)
	}
	m.gamesMutex.Lock()
	m.games = append(m.games, g)
	m.gamesMutex.Unlock()
	return nil
}

func skipContains(game interfaces.Game, skip []interfaces.Game) bool {
	for _, g := range skip {
		if g == game {
			return true
		}
	}
	return false
}

// ForEachGame allows you to run a function on every game the Manager manages. Note that this read locks the mutex
// protecting the games list. Any panics that occur will be recovered and logged as an error on the bot
func (m *Manager) ForEachGame(gameFunc func(interfaces.Game), skip []interfaces.Game) {
	var i int
	var g interfaces.Game
	defer func() {
		if err := recover(); err != nil {
			m.Error(fmt.Errorf("recovered a panic from function %p in ForEachGame on game %d (%s): %s", gameFunc, i, g, err))
		}
	}()
	m.gamesMutex.RLock()
	defer m.gamesMutex.RUnlock()
	for i, g = range m.games {
		if skipContains(g, skip) {
			continue
		}
		gameFunc(g)
	}
}

// StopAllGames stops all the games on the manager, blocking until they all close or are killed
func (m *Manager) StopAllGames() {
	m.status = shutdown
	wg := sync.WaitGroup{}
	m.ForEachGame(func(game interfaces.Game) { wg.Add(1); game.StopOrKillWaitgroup(&wg) }, nil)
	wg.Wait()
}

// StartAutoStartGames starts any game marked as auto start
func (m *Manager) StartAutoStartGames() {
	m.ForEachGame(func(game interfaces.Game) { game.AutoStart() }, nil)
}

func (m *Manager) StartGame(name string) error {
	if g := m.GetGameFromName(name); g != nil {
		go g.Run()
		return nil
	}
	return fmt.Errorf("game %q does not exist", name)
}

// Error is a helper function that returns the passed error to the manager's bot instance
func (m *Manager) Error(err error) {
	m.bot.Error(fmt.Errorf("game.Manager: %s", err))
}

func (m *Manager) setupCommands() {
	_ = m.bot.HookCommand("STARTGAME", 2, "starts the specified game",
		func(fromIRC bool, args []string, source ircutils.UserHost, target string, resp interfaces.CommandResponder) {
			if len(args) < 1 {
				resp.ReturnNotice("you must provide at least one game to start")
				return
			}
			for _, name := range args {
				if err := m.StartGame(name); err != nil {
					resp.ReturnNotice(err.Error())
				}
			}
		})
}
