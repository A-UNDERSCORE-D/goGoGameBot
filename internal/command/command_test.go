package command

import (
	"reflect"
	"testing"

	"github.com/goshuirc/irc-go/ircutils"
)

func TestSingleCommand_Fire(t *testing.T) {
	type fields struct {
		adminRequired int
		callback      Callback
		help          string
		name          string
	}
	type args struct {
		data Data
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &SingleCommand{
				adminRequired: tt.fields.adminRequired,
				callback:      tt.fields.callback,
				help:          tt.fields.help,
				name:          tt.fields.name,
			}
			c.Fire(tt.args.data)
		})
	}
}

func TestSingleCommand_AdminRequired(t *testing.T) {
	tests := []struct {
		name          string
		adminRequired int
	}{
		{
			name:          "basic test",
			adminRequired: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&SingleCommand{adminRequired: tt.adminRequired}).AdminRequired(); got != tt.adminRequired {
				t.Errorf("SingleCommand.AdminRequired() = %v, want %v", got, tt.adminRequired)
			}
		})
	}
}

func TestSingleCommand_Help(t *testing.T) {
	tests := []struct {
		name    string
		helpMsg string
	}{
		{"single case", "help for command is helpful"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&SingleCommand{help: tt.helpMsg}).Help(); got != tt.helpMsg {
				t.Errorf("SingleCommand.Help() = %v, want %v", got, tt.helpMsg)
			}
		})
	}
}

func TestSingleCommand_Name(t *testing.T) {
	testName := "test name is testy"
	if res := (&SingleCommand{name: testName}).Name(); res != testName {
		t.Errorf("SingleCommand.Name() = %q, want %q", res, testName)
	}
}

func TestSubCommandList_Help(t *testing.T) {
	testHelp := "test help is helpful"
	if res := (&SubCommandList{SingleCommand: SingleCommand{help: testHelp}}).Help(); res != testHelp {
		t.Errorf("SubCommandList.Help() = %v, want %v", res, testHelp)
	}
}

func TestSubCommandList_findSubcommand(t *testing.T) {
	baseSC := SingleCommand{
		help: "some help",
		name: "test",
	}
	type fields struct {
		subCommands map[string]Command
	}
	tests := []struct {
		name    string
		fields  fields
		cmdName string
		want    Command
	}{
		{
			name:    "nonexistant command",
			fields:  fields{map[string]Command{"tesasdt": &SingleCommand{name: "test", help: "some help"}}},
			cmdName: "test",
			want:    nil,
		},
		{
			name:    "command that exists",
			fields:  fields{map[string]Command{"test": &SingleCommand{name: "test", help: "some help"}}},
			cmdName: "test",
			want:    &SingleCommand{name: "test", help: "some help"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SubCommandList{
				SingleCommand: baseSC,
				subCommands:   tt.fields.subCommands,
			}
			if got := s.findSubcommand(tt.cmdName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SubCommandList.findSubcommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSubCommandList_addSubcommand(t *testing.T) {
	type fields struct {
		subCommands map[string]Command
	}
	type args struct {
		command Command
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "add",
			fields: fields{},
			args: args{
				command: &SingleCommand{name: "test"},
			},
			wantErr: false,
		},
		{
			name:    "add existing",
			fields:  fields{map[string]Command{"test": &SingleCommand{}}},
			args:    args{command: &SingleCommand{name: "test"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SubCommandList{
				SingleCommand: SingleCommand{},
				subCommands:   tt.fields.subCommands,
			}
			if err := s.addSubcommand(tt.args.command); (err != nil) != tt.wantErr {
				t.Errorf("SubCommandList.addSubcommand() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSubCommandList_Fire(t *testing.T) {
	manager := NewManager(baseLogger, &mockMessager{})
	type fields struct {
		subCommands map[string]Command
	}
	type args struct {
		data Data
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantMessage [][2]string
		wantNotice  [][2]string
	}{
		{
			name: "existing",
			fields: fields{map[string]Command{
				"test": &SingleCommand{
					name:     "test",
					callback: func(data Data) { data.SendSourceNotice("works") },
				},
			}},
			args: args{
				data: Data{
					IsFromIRC: true,
					Args:      []string{"test", "stuff"},
					Source:    ircutils.ParseUserhost("test!test@test"),
					Manager:   manager,
				},
			},
			wantNotice: [][2]string{{"test", "works"}},
		},
		{
			name: "does not exist",
			args: args{
				data: Data{
					IsFromIRC: true,
					Args:      []string{"test", "stuff"},
					Source:    ircutils.ParseUserhost("test!test@test"),
					Manager:   manager,
				},
			},
			wantNotice: [][2]string{{"test", "unknown subcommand \"test\""}, {"test", "Available subcommands are: "}},
		},
		{
			name: "not enough args",
			args: args{
				data: Data{
					IsFromIRC: true,
					Args:      []string{},
					Source:    ircutils.ParseUserhost("test!test@test"),
					Manager:   manager,
				},
			},
			wantNotice: [][2]string{{"test", "Not enough arguments"}, {"test", "Available subcommands are: "}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msger := tt.args.data.Manager.messenger.(*mockMessager)
			msger.Clear()
			s := &SubCommandList{
				SingleCommand: SingleCommand{},
				subCommands:   tt.fields.subCommands,
			}
			s.Fire(tt.args.data)
			if !cmpSlice(msger.lastMessages, tt.wantMessage) {
				t.Errorf("SubCommandList.Fire() sent messages %s, want %s", msger.lastMessages, tt.wantMessage)
			}
			if !cmpSlice(msger.lastNotices, tt.wantNotice) {
				t.Errorf("SubCommandList.Fire() sent notices %s, want %s", msger.lastNotices, tt.wantNotice)
			}
		})
	}
}
