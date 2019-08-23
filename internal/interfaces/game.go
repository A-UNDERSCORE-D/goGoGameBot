package interfaces

import (
	"io"
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
)

// GameManager handles games
type GameManager interface {
	ReloadGames(configs []config.Game)
	GetGameFromName(name string) Game
	GameExists(name string) bool
	AddGame(game Game) error
	ForEachGame(f func(Game), skip []Game)
	StopAllGames()
}

// Game Represents a runnable game server
type Game interface {
	GetName() string
	UpdateFromConfig(config.Game) error
	StopOrKiller
	Runner
	AutoStarter
	Statuser
	io.Writer
	io.StringWriter

	OnPrivmsg(source, target, msg string)
	OnJoin(source, channel string)
	OnPart(source, channel, message string)
	OnNick(source, newnick string)
	OnQuit(source, message string)
	OnKick(source, channel, kickee, message string)
	SendLineFromOtherGame(msg string, source Game)
}

// StopOrKiller holds methods to stop running processes, killing them after a timeout
type StopOrKiller interface {
	StopOrKill() error
	StopOrKillTimeout(duration time.Duration) error
	StopOrKillWaitgroup(group *sync.WaitGroup)
}

// Runner holds methods to Run a process and query the status
type Runner interface {
	Run()
	IsRunning() bool
}

// AutoStarter refers to any type that can be autostarted
type AutoStarter interface {
	AutoStart()
}

// Statuser refers to any type that can report its status as a string
type Statuser interface {
	Status() string
}
