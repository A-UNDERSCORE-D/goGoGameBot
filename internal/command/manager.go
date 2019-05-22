package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

const noAdmin = 0

func NewManager(logger *log.Logger, messenger interfaces.IRCMessager) *Manager {
	return &Manager{logger: logger, messenger: messenger, commands:make(map[string]Command)}
}

type Manager struct {
	admins    []Admin
	commands  map[string]Command
	logger    *log.Logger
	messenger interfaces.IRCMessager
}

func (m *Manager) AddCommand(name string, requiresAdmin int, callback Callback, help string) error {
	return m.addCommand(&SingleCommand{
		adminRequired: requiresAdmin,
		callback:      callback,
		help:          help,
		name:          name,
	})
}

func (m *Manager) addCommand(cmd Command) error {
	if strings.Contains(cmd.Name(), " ") {
		return errors.New("commands cannot contain spaces")
	}

	if m.getCommandByName(cmd.Name()) != nil {
		return fmt.Errorf("command %q already exists on %v", cmd.Name(), m)
	}
	m.commands[strings.ToLower(cmd.Name())] = cmd
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
	if c, ok := m.commands[strings.ToLower(name)]; ok {
		return c
	}
	return nil
}

func (m *Manager) AddSubCommand(rootName, name string, requiresAdmin int, callback Callback, help string) error {
	if m.getCommandByName(rootName) == nil {
		err := m.addCommand(&SubCommandList{
			SingleCommand: SingleCommand{adminRequired: noAdmin, callback: nil, help: "", name: rootName},
			subCommands:   nil,
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

func (m *Manager) ParseLine(line string, fromIRC bool, source ircutils.UserHost, target string) {
	if len(line) == 0 {
		return
	}
	lineSplit := strings.Split(line, " ")
	if len(lineSplit) < 1 {
		return
	}
	cmdName := lineSplit[0]
	cmd := m.getCommandByName(cmdName)
	if cmd == nil {
		return
	}

	data := Data{
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
	for k, _ := range m.commands {
		cmds = append(cmds, k)
	}
	var admins []string
	for _, v := range m.admins {
		admins = append(admins, fmt.Sprintf("%s: %d", v.Mask, v.Level))
	}
	return fmt.Sprintf("command.Manager containing commands: %s. and admins: %s", strings.Join(cmds, ", "), strings.Join(admins, ", "))
}
