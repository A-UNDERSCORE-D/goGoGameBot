package command

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

const noAdmin = 0

func NewManager(logger *log.Logger, messenger interfaces.IRCMessager, prefixes ...string) *Manager {
	m := &Manager{Logger: logger, messenger: messenger, commands: make(map[string]Command), commandPrefixes: prefixes}
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
		if data.IsFromIRC {
			data.SendSourceNotice(toSend)
		} else {
			m.Logger.Info(toSend)
		}
	}, "prints command help")
	return m
}

type Manager struct {
	admins          []Admin
	cmdMutex        sync.RWMutex
	commands        map[string]Command
	commandPrefixes []string
	Logger          *log.Logger
	messenger       interfaces.IRCMessager
}

func (m *Manager) AddPrefix(name string) {
	m.commandPrefixes = append(m.commandPrefixes, name)
}

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

func (m *Manager) AddCommand(name string, requiresAdmin int, callback Callback, help string) error {
	return m.addCommand(&SingleCommand{
		adminRequired: requiresAdmin,
		callback:      callback,
		help:          help,
		name:          strings.ToLower(name),
	})
}

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

func (m *Manager) addCommand(cmd Command) error {
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

func (m *Manager) AddAdmin(mask string, level int) error {
	if level <= 0 {
		return fmt.Errorf("admin level cannot be below 1 (0 is no access)")
	}
	for _, v := range m.admins {
		if v.Mask == mask {
			return fmt.Errorf("admin with mask %q already exists", mask)
		}
	}
	m.admins = append(m.admins, Admin{level, mask})
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

func (m *Manager) AddSubCommand(rootName, name string, requiresAdmin int, callback Callback, help string) error {
	if m.getCommandByName(rootName) == nil {
		err := m.addCommand(&SubCommandList{
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

func (m *Manager) adminFromMask(mask string) int {
	max := 0
	for _, admin := range m.admins {
		if admin.MatchesMask(mask) && admin.Level > max {
			max = admin.Level
		}
	}
	return max
}

const notAllowed = "You are not permitted to use this command"

func (m *Manager) CheckAdmin(data *Data, requiredLevel int) bool {
	if !data.IsFromIRC {
		return true // Non IRC users have direct access
	}
	if m.adminFromMask(data.SourceMask()) >= requiredLevel {
		return true
	}
	m.messenger.SendNotice(data.Source.Nick, notAllowed)
	return false
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

func (m *Manager) ParseLine(line string, fromIRC bool, source ircutils.UserHost, target string) {
	if len(line) == 0 {
		return
	}

	if fromIRC {
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
		if !fromIRC {
			m.Logger.Infof("unknown command %q", cmdName)
		}
		return
	}
	m.Logger.Debugf("firing command %q (original line %q)", cmdName, line)

	data := &Data{
		IsFromIRC:    fromIRC,
		Args:         lineSplit[1:],
		OriginalArgs: line,
		Source:       source,
		Target:       target,
		Manager:      m,
	}
	cmd.Fire(data)
}

func (m *Manager) String() string {
	var cmds []string
	m.cmdMutex.RLock()
	for k, _ := range m.commands {
		cmds = append(cmds, k)
	}
	m.cmdMutex.RUnlock()
	var admins []string
	for _, v := range m.admins {
		admins = append(admins, fmt.Sprintf("%s: %d", v.Mask, v.Level))
	}
	return fmt.Sprintf("command.Manager containing commands: %s. and admins: %s", strings.Join(cmds, ", "), strings.Join(admins, ", "))
}
