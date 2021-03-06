package game

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"awesome-dragon.science/go/goGoGameBot/internal/command"
	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
	"awesome-dragon.science/go/goGoGameBot/pkg/mutexTypes"
)

// NewManager creates a Manager and configures it using the given data.
func NewManager(conf *tomlconf.Config, bot interfaces.Bot, logger *log.Logger) (*Manager, error) {
	m := &Manager{
		bot:      bot,
		Logger:   logger.Clone().SetPrefix("GM"),
		done:     sync.NewCond(new(sync.Mutex)),
		rootConf: conf,
	}

	m.Cmd = command.NewManager(logger.Clone().SetPrefix("CMD"), bot.IsCommandPrefix, bot.StaticCommandPrefixes()...)
	m.setupHooks()
	m.ReloadGames(conf.Games)

	if err := m.setupCommands(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Manager) setupHooks() {
	m.bot.HookMessage(func(source, channel, message string, _ bool) {
		m.Cmd.ParseLine(message, false, source, channel, m.bot)
	})

	m.bot.HookMessage(func(source, channel, message string, isAction bool) {
		m.ForEachGame(func(game interfaces.Game) { game.OnMessage(source, channel, message, isAction) }, nil)
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
}

// Manager manages games, and communication between them, eachother, and an interfaces.Bot
type Manager struct {
	rootConf     *tomlconf.Config
	games        []interfaces.Game
	gamesMutex   sync.RWMutex
	bot          interfaces.Bot
	reconnecting mutexTypes.Bool
	Cmd          *command.Manager
	done         *sync.Cond
	restarting   mutexTypes.Bool
	status       mutexTypes.Int
	*log.Logger
}

// Run starts the manager, connects its bots
func (m *Manager) Run() (bool, error) {
	go m.runBot()
	go func() {
		time.Sleep(time.Second * 5)
		m.StartAutoStartGames()
	}()

	m.done.L.Lock()
	for m.status.Get() == normal {
		m.done.Wait()
	}
	m.done.L.Unlock()
	// Make sure we return a restart condition here if we need to
	return m.restarting.Get(), nil
}

func (m *Manager) runBot() {
	for {
		if err := m.bot.Run(); err != nil {
			m.Warnf("error occurred while running bot %s: %s", m.bot, err)
			m.Info("Sleeping for 1s")
			time.Sleep(time.Second * 1)
		}

		if m.status.Get() != normal {
			break
		}

		if !m.reconnecting.Get() {
			m.reconnecting.Set(false)
			m.sendStatusMessageToAllGames("Chat is disconnected. Reconnecting in 10 seconds")
			time.Sleep(time.Second * 10)
		} else {
			m.sendStatusMessageToAllGames("Chat is disconnected due to a reconnect request")
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func (m *Manager) sendStatusMessageToAllGames(msg string) {
	m.ForEachGame(func(g interfaces.Game) {
		g.SendLineFromOtherGame(msg, g)
	}, nil)
}

func (m *Manager) String() string {
	m.gamesMutex.RLock()
	defer m.gamesMutex.RUnlock()

	return fmt.Sprintf("game.Manager at %p with %d games attached", m, len(m.games))
}

// ReloadGames uses the passed config values to reload the games stored on it. Any new games
// found in the config are added, rather than reloaded
func (m *Manager) ReloadGames(configs []*tomlconf.Game) {
	// No need to hold the games mutex as of yet as we're not iterating the games list itself
	m.Debug("reloading games")
	defer m.Debug("games reload complete")

	for _, gameConf := range configs {
		switch internalIdx := m.gameIdxFromName(gameConf.Name); internalIdx {
		case -1: // Game does not exist
			m.Debugf("adding a new game during reload: %s", gameConf.Name)

			g, err := NewGame(gameConf, m)
			if err != nil {
				m.Error(err)
				continue
			}

			if err := m.AddGame(g); err != nil {
				m.Error(err)
				continue
			}
		default:
			m.Debugf("updating config on %s", gameConf.Name)
			// use RLock here because we're only reading the slice, and mutating an index on that slice
			m.gamesMutex.RLock()

			g := m.games[internalIdx]
			if err := g.UpdateFromConfig(gameConf); err != nil {
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
	var (
		i int
		g interfaces.Game
	)

	defer func() {
		if err := recover(); err != nil {
			m.Error(
				fmt.Errorf(
					"recovered a panic from function %p in ForEachGame on game %d (%s): %s", gameFunc, i, g, err,
				),
			)
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
	m.status.Set(shutdown)

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

		go func() { _ = g.Run() }()

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

		shutdownHelp = "shuts down the running bot instance, disconnects all connections, and stops all games"
		restartMHelp = "stops the running bot instance, disconnects all connections, and stops all games, " +
			"and then starts it all back up"
		reloadHelp = "reloads the config file from disk and applies it to the running bot." +
			" Note that some configuration changes require a restart of the bot"

		statusHelp = "returns the status of the bot. If a list of games is provided as arguments, " +
			"gets the status for each game. If all is provided as the first arg, all game's statuses are reported"

		reconnHelp = "reconnects the bot to the chat layer. "

		bot        = "bot"
		botRawHelp = "Sends a raw line directly to the chat platform in use"
	)

	var errs []error
	errs = append(
		errs,
		m.Cmd.AddSubCommand(gamectl, "start", 2, m.startGameCmd, startHelp),
		m.Cmd.AddSubCommand(gamectl, "stop", 2, m.stopGameCmd, stopHelp),
		m.Cmd.AddSubCommand(gamectl, "raw", 3, m.rawGameCmd, rawHelp),
		m.Cmd.AddSubCommand(gamectl, "restart", 2, m.restartGameCmd, restartHelp),
		m.Cmd.AddCommand("shutdown", 3, m.shutdownCmd, shutdownHelp),
		m.Cmd.AddCommand("restart", 3, m.restartCmd, restartMHelp),
		m.Cmd.AddCommand("reload", 3, m.reloadCmd, reloadHelp),
		m.Cmd.AddCommand("status", 0, m.statusCmd, statusHelp),
		m.Cmd.AddCommand("reconnect", 3, m.reconnectCmd, reconnHelp),
		m.Cmd.AddSubCommand(bot, "raw", 3, m.rawCmd, botRawHelp),
	)

	outErr := strings.Builder{}

	for _, e := range errs {
		if e != nil {
			outErr.WriteString(e.Error())
			outErr.WriteString(", ")
		}
	}

	if outErr.Len() > 0 {
		m.Warnf("init of static commands errored. THIS IS A BUG! REPORT IT!: %s", outErr.String())
		return errors.New(outErr.String())
	}

	return nil
}

// Stop stops all running games on the manager and disconnects the bot.
func (m *Manager) Stop(msg string, restart bool) {
	m.restarting.Set(restart)
	m.status.Set(shutdown)
	m.StopAllGames()
	m.bot.Disconnect(msg)
	m.done.Broadcast()
}

func (m *Manager) reload(conf *tomlconf.Config) error {
	m.rootConf = conf
	m.ReloadGames(conf.Games)

	// TODO: ensure that type wasn't changed
	if err := m.bot.Reload(conf.Connection.RealConf); err != nil {
		return err
	}

	m.Cmd.SetPrefixes(m.bot.StaticCommandPrefixes())

	return nil
}
