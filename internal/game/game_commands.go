package game

import (
	"errors"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/command"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/format"
)

func (g *Game) createCommandCallback(fmt format.Format) command.Callback {
	return func(data *command.Data) {
		res, err := fmt.ExecuteBytes(data)
		if err != nil {
			g.manager.Error(err)
			return
		}
		if _, err := g.Write(res); err != nil {
			g.manager.Error(err)
		}
	}
}

func (g *Game) registerCommand(conf config.Command) error {
	if conf.Name == "" {
		return errors.New("cannot have a game command with an empty name")
	}
	if conf.Help == "" {
		return errors.New("cannot have a game command with an empty help string")
	}
	if err := conf.Format.Compile(conf.Name, false, nil); err != nil {
		return err
	}
	return g.manager.Cmd.AddSubCommand(
		g.name,
		conf.Name,
		conf.RequiresAdmin,
		g.createCommandCallback(conf.Format),
		conf.Help,
	)
}

func (g *Game) clearCommands() error {
	return g.manager.Cmd.RemoveCommand(g.name)
}
