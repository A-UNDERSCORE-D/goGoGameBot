package game

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/command"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

// NewManager creates a Manager and configures it using the given data.
func NewManager(conf config.GameManager, bot interfaces.Bot, logger *log.Logger) (*Manager, error) {
	m := &Manager{
		bot:    bot,
		Logger: logger.Clone().SetPrefix("GM"),
		done:   sync.NewCond(new(sync.Mutex)),
	}
	m.Cmd = command.NewManager(logger.Clone().SetPrefix("CMD"), "!!")

	m.bot.HookMessage(func(source, channel, message string) {
		m.Cmd.ParseLine(message, false, source, channel, m.bot)
	})

	m.bot.HookMessage(func(source, channel, message string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnPrivmsg(source, channel, message) }, nil)
	})

	m.bot.HookJoin(func(source, channel string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnJoin(source, channel) }, nil)
	})

	m.bot.HookPart(func(source, channel, message string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnPart(source, channel, message) }, nil)
	})

	m.bot.HookQuit(func(source, message string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnQuit(source, message) }, nil)
	})

	m.bot.HookKick(func(source, channel, target, message string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnKick(source, channel, target, message) }, nil)
	})

	m.bot.HookNick(func(source, newNick string) {
		m.ForEachGame(func(game interfaces.Game) { game.OnNick(source, newNick) }, nil)
	})

	var games []interfaces.Game
	for _, g := range conf.Games {
		ng, err := NewGame(g, m)
		if err != nil {
			return nil, fmt.Errorf("could not create game %s: %s", g.Name, err)
		}
		games = append(games, ng)
	}
	m.games = games
	if err := m.setupCommands(); err != nil {
		return nil, err
	}

	return m, nil
}

// Manager manages games, and communication between them, eachother, and an interfaces.Bot
type Manager struct {
	games      []interfaces.Game
	gamesMutex sync.RWMutex
	bot        interfaces.Bot
	Cmd        *command.Manager
	status     status
	stripMasks []string
	done       *sync.Cond
	restarting bool
	*log.Logger
}

// Run starts the manager, connects its bots
func (m *Manager) Run() (bool, error) {
	go m.runBot()
	m.StartAutoStartGames()
	m.done.L.Lock()
	for m.status == normal {
		m.done.Wait()
	}
	m.done.L.Unlock()
	// Make sure we return a restart condition here if we need to
	return m.restarting, nil
}

func (m *Manager) runBot() {
	for {
		if err := m.bot.Run(); err != nil {
			m.Warnf("error occurred while running bot %s: %s", m.bot, err)
		}

		if m.status == normal {
			m.ForEachGame(func(g interfaces.Game) {
				g.SendLineFromOtherGame("Chat is disconnected. Reconnecting in 10 seconds", g)
			}, nil)
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}
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
	m.Debug("reloading games")
	for _, conf := range configs {
		switch i := m.gameIdxFromName(conf.Name); i {
		case -1: // Game does not exist
			g, err := NewGame(conf, m)
			m.Debugf("adding a new game during reload: %s", conf.Name)
			if err != nil {
				m.Error(err)
				continue
			}
			if err := m.AddGame(g); err != nil {
				m.Error(err)
				continue
			}
		default:
			m.Debugf("updating config on %s", conf.Name)
			m.gamesMutex.RLock() // use RLock here because we're only reading the slice, and mutating an index on that slice
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

var (
	// ErrGameNotExist is returned by various methods when the game requested does not exist
	ErrGameNotExist = errors.New("requested game does not exist")
)

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

// StartGame starts the game named if it exists on the manager and is not already running
func (m *Manager) StartGame(name string) error {
	if g := m.GetGameFromName(name); g != nil {
		if g.IsRunning() {
			return ErrAlreadyRunning
		}
		go g.Run()
		return nil
	}
	return ErrGameNotExist
}

// StopGame stops the named game on the manager if it exists and is already running
func (m *Manager) StopGame(name string) error {
	if g := m.GetGameFromName(name); g != nil {
		if !g.IsRunning() {
			return ErrGameNotRunning
		}

		if err := g.StopOrKill(); err != nil {
			return err
		}
	}
	return nil
}

// Error is a helper function that returns the passed error to the manager's bot instance
func (m *Manager) Error(err error) {
	m.bot.SendAdminMessage(fmt.Sprintf("game.Manager: %s", err))
	m.Logger.Warn(err)
	for _, l := range strings.Split(string(debug.Stack()), "\n") {
		m.Logger.Warn(l)
	}
}

func (m *Manager) setupCommands() error {
	const (
		gamectl     = "gamectl"
		startHelp   = "starts the provided games"
		stopHelp    = "stops the provided games, killing them if needed"
		restartHelp = "restarts the specified games, as with stop, games may be killed if a stop times out"
		rawHelp     = "sends the arguments provided directly to the standard in of the running game"

		stopMHelp    = "stops the running bot instance, disconnects all connections, and stops all games"
		restartMHelp = "stops the running bot instance, disconnects all connections, and stops all games, and then starts it all back up"
	)

	var err error
	err = m.Cmd.AddSubCommand(gamectl, "start", 2, m.startGameCmd, startHelp)
	err = m.Cmd.AddSubCommand(gamectl, "stop", 2, m.stopGameCmd, stopHelp)
	err = m.Cmd.AddSubCommand(gamectl, "raw", 2, m.rawGameCmd, rawHelp)
	err = m.Cmd.AddSubCommand(gamectl, "restart", 2, m.restartGameCmd, restartHelp)
	err = m.Cmd.AddCommand("stop", 2, m.stopCmd, stopMHelp)
	err = m.Cmd.AddCommand("restart", 2, m.restartCmd, restartHelp)

	if err != nil {
		m.Warnf("init of static commands errored. THIS IS A BUG! REPORT IT!: %s", err)
		return err
	}

	return nil
}

func (m *Manager) Stop(msg string, restart bool) {
	m.restarting = restart
	m.bot.SendAdminMessage(msg)
	m.status = shutdown
	m.StopAllGames()
	m.bot.Disconnect("Stop requested")
	m.done.Broadcast()
}
