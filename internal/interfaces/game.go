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

	OnPrivmsg(source, target, msg string)
	OnJoin(source, channel string)
	OnPart(source, channel, message string)
	OnNick(source, newnick string)
	OnQuit(source, message string)
	OnKick(source, channel, kickee, message string)
	SendLineFromOtherGame(msg string, source Game)
}

type StopOrKiller interface {
	StopOrKill() error
	StopOrKillTimeout(duration time.Duration) error
	StopOrKillWaitgroup(group *sync.WaitGroup)
}

type Runner interface {
	Run()
	IsRunning() bool
}

type AutoStarter interface {
	AutoStart()
}

type Statuser interface {
	Status() string
}
