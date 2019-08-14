package command

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

var (
	baseLogger = log.New(log.FTimestamp|log.FShowFile, os.Stdout, "TEST", log.INFO)
)

var _ DataUtil = &mockMessager{}

type mockMessager struct {
	lastMessages [][2]string
	lastNotices  [][2]string
	lastRaw      []string
	admins       map[string]int
}

func getNick(source string) string {
	if strings.HasPrefix(source, "#") {
		return source
	}
	return ircutils.ParseUserhost(source).Nick
}

func (m *mockMessager) SendMessage(target, message string) {
	m.lastMessages = append(m.lastMessages, [2]string{getNick(target), message})
}

func (m *mockMessager) SendNotice(target, message string) {
	m.lastNotices = append(m.lastNotices, [2]string{getNick(target), message})
}

/*func (m *mockMessager) WriteString(message string) error {
	m.lastRaw = append(m.lastRaw, message)
	return nil
}*/

func (m *mockMessager) checkMap() {
	if m.admins == nil {
		m.admins = make(map[string]int)
	}
}

func (m *mockMessager) AdminLevel(source string) int {
	m.checkMap()
	for mask, level := range m.admins {
		if util.GlobToRegexp(mask).MatchString(source) {
			return level
		}
	}
	return 0
}

func (m *mockMessager) AddAdmin(mask string, level int) {
	m.checkMap()
	m.admins[mask] = level
}

func cmpSlice(a, b [][2]string) bool {
	if len(b) != len(a) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	for i, v := range a {
		if b[i][0] != v[0] || b[i][1] != v[1] {
			return false
		}
	}

	return true
}

func (m *mockMessager) Clear() {
	*m = mockMessager{}
}

func TestManager_AddCommand(t *testing.T) {
	preExisting := NewManager(baseLogger)
	_ = preExisting.AddCommand("dupe", 0, nil, "dupe command is duped")
	type args struct {
		name          string
		requiresAdmin int
		help          string
	}
	tests := []struct {
		name    string
		m       *Manager
		args    args
		wantErr bool
	}{
		{
			name: "space error",
			m:    NewManager(baseLogger),
			args: args{
				name:          "basic test",
				requiresAdmin: 0,
				help:          "",
			},
			wantErr: true,
		},
		{
			name: "singlename test",
			m:    NewManager(baseLogger),
			args: args{
				name:          "test",
				requiresAdmin: 0,
				help:          "here have a test command",
			},
			wantErr: false,
		},
		{
			name: "duped commands",
			m:    preExisting,
			args: args{
				"DuPe",
				0,
				"duped command is duped",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.m.AddCommand(tt.args.name, tt.args.requiresAdmin, nil, tt.args.help); (err != nil) != tt.wantErr {
				t.Errorf("Manager.AddCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_getCommandByName(t *testing.T) {
	m := NewManager(baseLogger)
	existingCommand := &SingleCommand{
		0,
		nil,
		"Helpful help is helpful",
		"helpful",
	}
	existingSubCommand := &SubCommandList{
		SingleCommand: SingleCommand{0, nil, "test is not doing, allah is doing", "test"},
		subCommands:   map[string]Command{"test": &SingleCommand{0, nil, "lol", "test"}},
	}
	_ = m.internalAddCommand(existingCommand)
	_ = m.internalAddCommand(existingSubCommand)
	tests := []struct {
		name    string
		cmdName string
		want    Command
	}{
		{
			"existing command",
			"helpful",
			existingCommand,
		},
		{
			"existing command subcmd",
			"test",
			existingSubCommand,
		},
		{
			"nonexistent command",
			"nonexistent",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.getCommandByName(tt.cmdName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Manager.getCommandByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_AddSubCommand(t *testing.T) {
	sCmdManager := NewManager(baseLogger)
	_ = sCmdManager.internalAddCommand(&SingleCommand{0, nil, "single_command", "single"})
	mCmdManager := NewManager(baseLogger)
	_ = mCmdManager.internalAddCommand(&SubCommandList{
		SingleCommand: SingleCommand{0, nil, "baseCmd", "baseCmd"},
		subCommands:   make(map[string]Command)},
	)
	type args struct {
		rootName      string
		name          string
		requiresAdmin int
		callback      Callback
		help          string
	}
	tests := []struct {
		name    string
		m       *Manager
		args    args
		wantErr bool
	}{
		{
			name: "no existing root",
			m:    NewManager(baseLogger),
			args: args{
				rootName: "IDontExist",
				name:     "test",
			},
			wantErr: false,
		},
		{
			name: "bad root",
			m:    sCmdManager,
			args: args{
				rootName: "single",
				name:     "test",
			},
			wantErr: true,
		},
		{
			name: "existing root",
			m:    mCmdManager,
			args: args{
				rootName:      "baseCmd",
				name:          "test",
				requiresAdmin: 0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.m.AddSubCommand(tt.args.rootName, tt.args.name, tt.args.requiresAdmin, tt.args.callback, tt.args.help); (err != nil) != tt.wantErr {
				t.Errorf("Manager.AddSubCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func makeDataWithSourceAndUtil(mask string, dataUtil DataUtil) *Data {
	return &Data{Source: mask, FromTerminal: false, util: dataUtil}
}

/*
TODO: replace with TestData_checkAdmin or similar
func TestManager_CheckAdmin(t *testing.T) {
	m := NewManager(baseLogger)
	msg := &mockMessager{}
	msg.AddAdmin("*!*@someHost", 1)
	msg.AddAdmin("*!test@*", 2)
	zeroAccessUser := makeDataWithSourceAndUtil("unimportant!user@nowhere", msg)
	tests := []struct {
		name             string
		required         int
		data             *Data
		want             bool
		expectedNotices  [][2]string
		expectedMessages [][2]string
	}{
		{
			name:     "zero is zero",
			required: 0,
			data:     zeroAccessUser,
			want:     true,
		},
		{
			name:            "too low a level",
			required:        1337,
			data:            zeroAccessUser,
			expectedNotices: [][2]string{{zeroAccessUser.Source, notAllowed}},
			want:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			realMessanger := tt.data.util.(*mockMessager)
			realMessanger.Clear()
			if got := m.CheckAdmin(tt.data, tt.required); got != tt.want {
				t.Errorf("Manager.CheckAdmin() = %v, want %v", got, tt.want)
			}
			if !cmpSlice(realMessanger.lastNotices, tt.expectedNotices) {
				t.Errorf("Manager.CheckAdmin() did not send expected notices. got %v, want %v", realMessanger.lastNotices, tt.expectedNotices)
			}
			if !cmpSlice(realMessanger.lastMessages, tt.expectedMessages) {
				t.Errorf("Manager.CheckAdmin() did not send expected messages. got %v, want %v", realMessanger.lastMessages, tt.expectedMessages)
			}
		})
	}
}*/

func TestManager_ParseLine(t *testing.T) {
	m := NewManager(baseLogger, "~")
	_ = m.AddCommand(
		"testNoAccess",
		noAdmin,
		func(data *Data) { data.SendTargetMessage("huzzah!") },
		"test cmd",
	)
	_ = m.AddCommand(
		"testaccess",
		1,
		func(data *Data) { data.SendTargetMessage("admin!") },
		"test cmd",
	)
	_ = m.AddSubCommand(
		"test",
		"cmdAccess",
		1,
		func(data *Data) { data.SendTargetMessage("HI! Im a subcommand that requires admin!") },
		"test cmd",
	)

	_ = m.AddSubCommand(
		"test",
		"cmdNoAccess",
		noAdmin,
		func(data *Data) { data.SendTargetMessage("HI! Im a subcommand that does not require admin") },
		"test cmd",
	)
	messager := &mockMessager{}
	type args struct {
		line         string
		fromTerminal bool
		source       string
		target       string
	}
	tests := []struct {
		name             string
		args             args
		expectedMessages [][2]string
		expectedNotices  [][2]string
	}{
		{
			name: "empty line",
			args: args{
				line:         "",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
		},
		{
			name: "normal call",
			args: args{
				line:         "~testNoAccess",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "huzzah!"}},
		},
		{
			name: "nonexistant call",
			args: args{
				line:         "~hi, I dont exist!",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
		},
		{
			name: "normal call weird case",
			args: args{
				line:         "~tEsTNOAcCeSs",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "huzzah!"}},
		},
		{
			name: "normal call with args",
			args: args{
				line:         "~testNoAccess except this time with arguments",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "huzzah!"}},
		},
		{
			name: "normal call but not from IRC",
			args: args{
				line:         "testNoAccess",
				fromTerminal: true,
			},
			expectedMessages: [][2]string{{"", "huzzah!"}}, // It tries to send a message anyway, but thats not our fault.
		},
		{
			name: "privileged call without access",
			args: args{
				line:         "~testAccess",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedNotices: [][2]string{{"test", notAllowed}},
		},
		{
			name: "privileged call with access",
			args: args{
				line:         "~testAccess",
				fromTerminal: false,
				source:       "picard!jean-luc@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "admin!"}},
			expectedNotices:  nil,
		},
		{
			name: "privileged call not from IRC",
			args: args{
				line:         "testAccess",
				fromTerminal: true,
			},
			expectedMessages: [][2]string{{"", "admin!"}},
		},
		{
			name: "nested normal",
			args: args{
				line:         "~test cmdNoAccess",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "HI! Im a subcommand that does not require admin"}},
		},
		{
			name: "nested privileged no access",
			args: args{
				line:         "~test cmdAccess",
				fromTerminal: false,
				source:       "test!test@test",
				target:       "#test",
			},
			expectedNotices: [][2]string{{"test", notAllowed}},
		},
		{
			name: "nested privileged access",
			args: args{
				line:         "~test cmdAccess",
				fromTerminal: false,
				source:       "picard!jean-luc@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "HI! Im a subcommand that requires admin!"}},
		},
		{
			name: "nested privileged access non IRC",
			args: args{
				line:         "test cmdAccess",
				fromTerminal: true,
			},
			expectedMessages: [][2]string{{"", "HI! Im a subcommand that requires admin!"}},
		},
		{
			name: "nested nonexistent",
			args: args{
				line:         "~test IDontExist",
				fromTerminal: false,
				source:       "picard!jean-luc@test",
				target:       "#test",
			},
			expectedNotices: [][2]string{
				{"picard", "unknown subcommand \"IDontExist\""},
				{"picard", "Available subcommands are: cmdAccess, cmdNoAccess"},
			},
		},
		{
			name: "nested weird casing",
			args: args{
				line:         "~test cMdNOAcCeSs",
				fromTerminal: false,
				source:       "picard!jean-luc@test",
				target:       "#test",
			},
			expectedMessages: [][2]string{{"#test", "HI! Im a subcommand that does not require admin"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messager.Clear()
			messager.AddAdmin("picard!jean-luc@*", 1337)
			m.ParseLine(tt.args.line, tt.args.fromTerminal, tt.args.source, tt.args.target, messager)
			if !cmpSlice(tt.expectedMessages, messager.lastMessages) {
				t.Errorf("Manager.Parse() did not send expected messages. got %v, want %v", messager.lastMessages, tt.expectedMessages)
			}
			if !cmpSlice(tt.expectedNotices, messager.lastNotices) {
				t.Errorf("Manager.Parse() did not send expected notices. got %v, want %v", messager.lastNotices, tt.expectedNotices)
			}
		})
	}
}

/*func hasAdmin(m *Manager, mask string, level int) bool {
	for _, a := range m.admins {
		if a.Level == level && mask == a.Mask {
			return true
		}
	}
	return false
}
*/
/*
	TODO: replace with an equiv for TestData
func TestManager_AddAdmin(t *testing.T) {
	m := NewManager(baseLogger, &mockMessager{})

	type args struct {
		mask  string
		level int
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"normal add", args{"*!*@test", 1}, false},
		{"duplicated add", args{"*!*@test", 1}, true},
		{"invalid level", args{"*!test@*", 0}, true},
		{"invalid level 2", args{"*!test2@*", -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := m.AddAdmin(tt.args.mask, tt.args.level); (err != nil) != tt.wantErr {
				t.Errorf("Manager.AddAdmin() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !hasAdmin(m, tt.args.mask, tt.args.level) {
				t.Errorf("Manager.AddAdmin() did not add admin correctly")
			}
		})
	}
}
*/
