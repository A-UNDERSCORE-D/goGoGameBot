package game

import (
	"errors"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

func (g *Game) createCommandCallback(fmt util.Format) interfaces.CmdFunc {
	return func(fromIRC bool, args []string, source ircutils.UserHost, target string) {
		res, err := fmt.ExecuteBytes(struct {
			FromIRC bool
			Args    []string
			Source  ircutils.UserHost
			Target  string
		}{fromIRC, args, source, target})
		if err != nil {
			g.manager.Error(err)
			return
		}
		if _, err := g.Write(res); err != nil {
			g.manager.Error(err)
		}
	}
}

func (g *Game) registerCommand(conf config.GameCommandConfig) error {
	if conf.Name == "" {
		return errors.New("cannot have a game command with an empty name")
	}
	if conf.Help == "" {
		return errors.New("cannot have a game command with an empty help string")
	}
	if err := conf.StdinFormat.Compile(g.name+"_"+conf.Name, false); err != nil {
		return err
	}
	return g.manager.bot.HookSubCommand(
		g.name,
		conf.Name,
		conf.RequiresAdmin,
		conf.Help,
		g.createCommandCallback(conf.StdinFormat),
	)
}
