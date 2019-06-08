package interfaces

import (
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
)

type GameManager interface {
	ReloadGames(configs []config.Game)
	GetGameFromName(name string) Game
	GameExists(name string) bool
	AddGame(game Game) error
	ForEachGame(f func(Game), skip []Game)
	StopAllGames()
}

type Game interface {
	GetName() string
	UpdateFromConfig(config config.Game) error
	WriteExternalMessage(msg string) error
	StopOrKiller
	Runner
	AutoStarter
	Statuser
}

type StopOrKiller interface {
	StopOrKill() error
	StopOrKillTimeout(duration time.Duration) error
	StopOrKillWaitgroup(group *sync.WaitGroup)
}

type Runner interface {
	Run()
}

type AutoStarter interface {
	AutoStart()
}

type Statuser interface {
	Status() string
}
