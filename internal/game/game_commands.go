package game

import (
	"errors"

	"awesome-dragon.science/go/goGoGameBot/internal/command"
	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/pkg/format"
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

func (g *Game) registerCommand(name string, conf tomlconf.Command) error {
	if name == "" {
		return errors.New("cannot have a game command with an empty name")
	}

	if conf.Help == "" {
		return errors.New("cannot have a game command with an empty help string")
	}

	fmt := format.Format{FormatString: conf.Format}

	if err := fmt.Compile(name, nil, nil); err != nil {
		return err
	}

	return g.manager.Cmd.AddSubCommand(
		g.name,
		name,
		conf.RequiresAdmin,
		g.createCommandCallback(fmt),
		conf.Help,
	)
}

func (g *Game) clearCommands() error {
	return g.manager.Cmd.RemoveCommand(g.name)
}
