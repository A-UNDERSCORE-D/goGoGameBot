package command

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

const noAdmin = 0

// TODO: have this require a function to use to check for admin level -- that way interfaces.bots can implement
//		 that on their own, and we just need to ask them what level a given source string has

// NewManager creates a Manager with the provided logger and messager. The prefixes vararg sets the prefixes for the
// commands. Note that the prefix is matched EXACTLY. Meaning that a trailing space is required for any "normal" prefix
func NewManager(logger *log.Logger, prefixes ...string) *Manager {
	m := &Manager{Logger: logger, commands: make(map[string]Command), commandPrefixes: prefixes}
	_ = m.AddCommand("help", 0, func(data *Data) {
		var toSend string
		if len(data.Args) == 0 {
			// just dump the available commands
			var commandNames []string
			m.cmdMutex.RLock()
			for _, c := range m.commands {
				commandNames = append(commandNames, c.Name())
			}
			m.cmdMutex.RUnlock()
			toSend = fmt.Sprintf("Available commands are %s", strings.Join(commandNames, ", "))
		} else {
			// specific help on a command requested
			var cmd Command
			if cmd = m.getCommandByName(data.Args[0]); cmd == nil {
				return
			}

			if realCmd, ok := cmd.(*SubCommandList); ok && len(data.Args) > 1 && realCmd.findSubcommand(data.Args[1]) != nil {
				subCmd := realCmd.findSubcommand(data.Args[1])
				toSend = fmt.Sprintf("%s: %s", strings.Join(data.Args[:2], " "), subCmd.Help())
			} else {
				toSend = fmt.Sprintf("%s: %s", data.Args[0], cmd.Help())
			}
		}
		if data.FromTerminal {
			m.Logger.Info(toSend)
		} else {
			data.SendSourceNotice(toSend)
		}
	}, "prints command help")
	return m
}

// Manager is a frontend that manages commands and the firing thereof. It is intended to be a completely self contained
// system for managing commands on arbitrary lines
type Manager struct {
	cmdMutex        sync.RWMutex
	commands        map[string]Command
	commandPrefixes []string
	Logger          *log.Logger
}

// AddPrefix adds a prefix to the command manager. It is not safe for concurrent use
func (m *Manager) AddPrefix(name string) {
	m.commandPrefixes = append(m.commandPrefixes, name)
}

// RemovePrefix removes a prefix from the Manager, it is not safe for concurrent use. If the prefix does not exist, the
// method is a noop
func (m *Manager) RemovePrefix(name string) {
	toRemove := -1
	for i, pfx := range m.commandPrefixes {
		if pfx == name {
			toRemove = i
			break
		}
	}
	if toRemove != -1 {
		m.commandPrefixes = append(m.commandPrefixes[:toRemove], m.commandPrefixes[toRemove+1:]...)
	}
}

// AddCommand adds the callback as a simple (SingleCommand) to the Manager. It is safe for concurrent use. It returns
// various errors
func (m *Manager) AddCommand(name string, requiresAdmin int, callback Callback, help string) error {
	return m.internalAddCommand(&SingleCommand{
		adminRequired: requiresAdmin,
		callback:      callback,
		help:          help,
		name:          strings.ToLower(name),
	})
}

// RemoveCommand removes the command referenced by the given string. If the command does not exist, RemoveCommand
// returns an error.
func (m *Manager) RemoveCommand(name string) error {
	if m.getCommandByName(name) == nil {
		return fmt.Errorf("command %q does not exist on %v", name, m)
	}
	m.Logger.Debugf("removing command %s", name)
	m.cmdMutex.Lock()
	defer m.cmdMutex.Unlock()
	delete(m.commands, name)
	return nil
}

// internalAddCommand adds the actual Command to the manager, it is used by both of the exported command addition methods
func (m *Manager) internalAddCommand(cmd Command) error {
	if strings.Contains(cmd.Name(), " ") {
		return errors.New("commands cannot contain spaces")
	}

	if m.getCommandByName(cmd.Name()) != nil {
		return fmt.Errorf("command %q already exists on %v", cmd.Name(), m)
	}
	m.Logger.Debugf("adding command %s: %v", cmd.Name(), cmd)
	m.cmdMutex.Lock()
	m.commands[strings.ToLower(cmd.Name())] = cmd
	m.cmdMutex.Unlock()
	return nil
}

func (m *Manager) getCommandByName(name string) Command {
	m.cmdMutex.RLock()
	defer m.cmdMutex.RUnlock()
	if c, ok := m.commands[strings.ToLower(name)]; ok {
		return c
	}
	return nil
}

// AddSubCommand adds the given callback as a subcommand to the given root name. If the root name does not exist on
// the Manager, it is automatically added. Otherwise, if it DOES exist but is of the wrong type, AddSubCommand returns
// an error
func (m *Manager) AddSubCommand(rootName, name string, requiresAdmin int, callback Callback, help string) error {
	if m.getCommandByName(rootName) == nil {
		err := m.internalAddCommand(&SubCommandList{
			SingleCommand: SingleCommand{adminRequired: noAdmin, callback: nil, help: "", name: strings.ToUpper(rootName)},
			subCommands:   make(map[string]Command),
		})
		if err != nil {
			return err
		}
	}

	var cmd *SubCommandList
	var ok bool
	if cmd, ok = m.getCommandByName(rootName).(*SubCommandList); !ok {
		return fmt.Errorf("command %s is not a command that can have subcommands", rootName)
	}
	return cmd.addSubcommand(&SingleCommand{name: name, adminRequired: requiresAdmin, callback: callback, help: help})
}

// RemoveSubCommand removes the command referenced by name on rootName, if rootName is not a command with sub commands,
// or name does not exist on rootName, RemoveSubCommand errors
func (m *Manager) RemoveSubCommand(rootName, name string) error {
	var cmd Command
	if cmd = m.getCommandByName(rootName); cmd == nil {
		return fmt.Errorf("command %q does not exist on %v", rootName, m)
	}
	var realCmd *SubCommandList
	var ok bool
	if realCmd, ok = cmd.(*SubCommandList); !ok {
		return fmt.Errorf("command %q is not a command that has subcommands", rootName)
	}

	return realCmd.removeSubcmd(name)
}

func (m *Manager) stripPrefix(line string) (string, bool) {
	hasPrefix := false
	var out string
	for _, pfx := range m.commandPrefixes {
		if strings.HasPrefix(strings.ToUpper(line), strings.ToUpper(pfx)) {
			hasPrefix = true
			out = line[len(pfx):]
		}
	}

	return out, hasPrefix
}

// ParseLine checks the given string for a valid command. If it finds one, it fires that command.
func (m *Manager) ParseLine(line string, fromTerminal bool, source, target string, util DataUtil) {
	if len(line) == 0 {
		return
	}

	if !fromTerminal {
		var ok bool
		if line, ok = m.stripPrefix(line); !ok {
			return
		}
	}

	lineSplit := strings.Split(line, " ")
	if len(lineSplit) < 1 {
		return
	}

	cmdName := lineSplit[0]
	cmd := m.getCommandByName(cmdName)
	if cmd == nil {
		if fromTerminal {
			m.Logger.Infof("unknown command %q", cmdName)
		}
		return
	}

	data := &Data{
		FromTerminal: fromTerminal,
		Args:         lineSplit[1:],
		OriginalArgs: line,
		Source:       source,
		Target:       target,
		Manager:      m,
		util:         util,
	}
	cmd.Fire(data)
}

// String implements the stringer interface
func (m *Manager) String() string {
	var cmds []string
	m.cmdMutex.RLock()
	for k := range m.commands {
		cmds = append(cmds, k)
	}
	m.cmdMutex.RUnlock()
	return fmt.Sprintf("command.Manager containing commands: %s", strings.Join(cmds, ", "))
}
