// Package interfaces ...
//nolint:misspell // I know, BUT, I cant fix it
package interfaces

import (
	"io"
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
)

// GameManager handles games
type GameManager interface {
	ReloadGames(configs []config.Game) // Reload the games on this Manager with the given configs
	GetGameFromName(name string) Game  // get the Game instance on this Manager that has the given name, or nil
	GameExists(name string) bool       // check whether or not this Manager has a Game with this name
	AddGame(game Game) error           // add a Game to this manager (game names should be case sensitive and unique)
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
	Statuser //nolint:misspell // Its Status-er not a misspelling of stature
	io.Writer
	io.StringWriter

	OnMessage(source, target, msg string, isAction bool)
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
type Statuser interface { //nolint:misspell // Its Status-er not a misspelling of stature
	// Status returns a human readable status string
	Status() string
}
