package command

import (
	"fmt"
	"strings"
)

type Callback func(data Data)

type Command interface {
	AdminRequired() int
	Fire(data Data)
	Help() string
	Name() string
}
type SingleCommand struct {
	adminRequired int
	callback      Callback
	help          string
	name          string
}

func (c *SingleCommand) Fire(data Data) {
	if data.CheckPerms(c.adminRequired) {
		c.callback(data)
	}
}

func (c *SingleCommand) AdminRequired() int { return c.adminRequired }
func (c *SingleCommand) Help() string       { return c.help }
func (c *SingleCommand) Name() string       { return c.name }

type SubCommandList struct {
	SingleCommand
	subCommands map[string]Command
}

func (s *SubCommandList) Help() string {
	out := strings.Builder{}
	out.WriteString("Available subcommands are: ")
	var subCmds []string
	for _, c := range s.subCommands {
		subCmds = append(subCmds, c.Name())
	}
	out.WriteString(strings.Join(subCmds, ", "))
	return out.String()
}

func (s *SubCommandList) findSubcommand(name string) Command {
	if s.subCommands == nil {
		s.subCommands = make(map[string]Command)
	}
	if c, ok := s.subCommands[strings.ToLower(name)]; ok {
		return c
	}
	return nil
}

func (s *SubCommandList) addSubcommand(command Command) error {
	if s.findSubcommand(command.Name()) != nil {
		return fmt.Errorf("command %s already exists on command %s", command.Name(), s.Name())
	}
	s.subCommands[strings.ToLower(command.Name())] = command
	return nil
}

func (s *SubCommandList) Fire(data Data) {
	if len(data.Args) < 1 {
		data.SendSourceNotice("Not enough arguments")
		data.SendSourceNotice(s.Help())
		return
	}

	c := s.findSubcommand(data.Args[0])
	if c == nil {
		data.SendSourceNotice(fmt.Sprintf("unknown subcommand %q", data.Args[0]))
		data.SendSourceNotice(s.Help())
		return
	}

	newData := Data{
		IsFromIRC:    data.IsFromIRC,
		Args:         data.Args[0:],
		OriginalArgs: data.OriginalArgs,
		Source:       data.Source,
		Target:       data.Target,
		Manager:      data.Manager,
	}
	c.Fire(newData)
}
