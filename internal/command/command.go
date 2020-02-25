package command

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Callback is a function that can be attached to a command
type Callback func(data *Data)

// Command represents a fireable command
type Command interface {
	AdminRequired() int
	Fire(data *Data)
	Help() string
	Name() string
}

// SingleCommand represents a Command with no special subcommand magic. It implements the Command interface
type SingleCommand struct {
	adminRequired int
	callback      Callback
	help          string
	name          string
}

// Fire executes the callback on the command if the caller has the permissions required
func (c *SingleCommand) Fire(data *Data) {
	if data.CheckPerms(c.adminRequired) {
		c.callback(data)
	}
}

// AdminRequired is a getter for the admin required on the command
func (c *SingleCommand) AdminRequired() int { return c.adminRequired }

// Help is a getter for the help string for the command
func (c *SingleCommand) Help() string { return c.help }

// Name is a getter for the name of the command
func (c *SingleCommand) Name() string { return c.name }

// SubCommandList is an implementation of Command that holds a list of subCommands that it can fire
type SubCommandList struct {
	SingleCommand
	sync.RWMutex
	subCommands map[string]Command
}

// Help returns the help message for the command. For SubCommandLists, this returns the available subcommands
func (s *SubCommandList) Help() string {
	out := strings.Builder{}
	out.WriteString("Available subcommands are: ")

	var subCmds []string

	s.RLock()
	for _, c := range s.subCommands {
		subCmds = append(subCmds, c.Name())
	}
	s.RUnlock()
	sort.Strings(subCmds)
	out.WriteString(strings.Join(subCmds, ", "))

	return out.String()
}

func (s *SubCommandList) findSubcommand(name string) Command {
	s.RLock()
	if c, ok := s.subCommands[strings.ToLower(name)]; ok {
		return c
	}
	s.RUnlock()

	return nil
}

func (s *SubCommandList) addSubcommand(command Command) error {
	if s.findSubcommand(command.Name()) != nil {
		return fmt.Errorf("command %s already exists on command %s", command.Name(), s.Name())
	}

	s.Lock()
	s.subCommands[strings.ToLower(command.Name())] = command
	s.Unlock()

	return nil
}

func (s *SubCommandList) removeSubcmd(name string) error {
	cmd := s.findSubcommand(name)
	if cmd == nil {
		return fmt.Errorf("%q does not have a subcommand called %q", s.Name(), name)
	}

	s.Lock()
	delete(s.subCommands, name)
	s.Unlock()

	return nil
}

// Fire executes the callback on the command if the caller has the permissions required
func (s *SubCommandList) Fire(data *Data) {
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

	newData := &Data{
		FromTerminal: data.FromTerminal,
		Args:         data.Args[1:],
		OriginalArgs: data.OriginalArgs,
		Source:       data.Source,
		Target:       data.Target,
		Manager:      data.Manager,
		util:         data.util,
	}

	c.Fire(newData)
}
