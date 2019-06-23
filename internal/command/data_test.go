package command

import (
	"testing"

	"github.com/goshuirc/irc-go/ircutils"
)

func TestData_CheckPerms(t *testing.T) {
	// This is more thoroughly tested on the manager's side. Lets just make sure that it works in a simple case here
	d := Data{
		IsFromIRC:    true,
		Args:         []string{"this", "doesnt", "matter"},
		OriginalArgs: "this doesnt matter",
		Source:       ircutils.ParseUserhost("test!test@test"),
		Target:       "#test",
		Manager:      NewManager(baseLogger, &mockMessager{}),
	}
	_ = d.Manager.AddAdmin("*!*@test", 1)
	if d.CheckPerms(2) {
		t.Errorf("Data.CheckAdmin() = %v, want %v", true, false)
	}
}

func TestData_SendTargetNotice(t *testing.T) {
	d := Data{
		Source:  ircutils.ParseUserhost("test!test@test"),
		Target:  "#test",
		Manager: NewManager(baseLogger, &mockMessager{}),
	}
	d.SendTargetNotice("test")
	n := d.Manager.messenger.(*mockMessager).lastNotices
	want := [][2]string{{"#test", "test"}}
	if !cmpSlice(n, want) {
		t.Errorf("Data.SendTargetNotice() did not send expected data: got %v, want %v", n, want)
	}
}

func TestData_SendTargetMessage(t *testing.T) {
	d := Data{
		Source:  ircutils.ParseUserhost("test!test@test"),
		Target:  "#test",
		Manager: NewManager(baseLogger, &mockMessager{}),
	}
	d.SendTargetMessage("test")
	n := d.Manager.messenger.(*mockMessager).lastMessages
	want := [][2]string{{"#test", "test"}}
	if !cmpSlice(n, want) {
		t.Errorf("Data.SendTargetMessage() did not send expected data: got %v, want %v", n, want)
	}
}

func TestData_SendSourceNotice(t *testing.T) {
	d := Data{
		Source:  ircutils.ParseUserhost("test!test@test"),
		Target:  "#test",
		Manager: NewManager(baseLogger, &mockMessager{}),
	}
	d.SendSourceNotice("test message")
	n := d.Manager.messenger.(*mockMessager).lastNotices
	want := [][2]string{{"test", "test message"}}
	if !cmpSlice(n, want) {
		t.Errorf("Data.SendSourceNotice() did not send expected data: got %v, want %v", n, want)
	}
}

func TestData_SendSourceMessage(t *testing.T) {
	d := Data{
		Source:  ircutils.ParseUserhost("test!test@test"),
		Target:  "#test",
		Manager: NewManager(baseLogger, &mockMessager{}),
	}
	d.SendSourceMessage("test message")
	n := d.Manager.messenger.(*mockMessager).lastMessages
	want := [][2]string{{"test", "test message"}}
	if !cmpSlice(n, want) {
		t.Errorf("Data.SendSourceNotice() did not send expected data: got %v, want %v", n, want)
	}
}

func TestData_SourceMask(t *testing.T) {
	tests := []struct {
		name string
		mask string
	}{
		{
			"basic test",
			"test!testident@testhost",
		},
		{
			"missing ident",
			"test@host",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Data{
				Source: ircutils.ParseUserhost(tt.mask),
			}
			if got := d.SourceMask(); got != tt.mask {
				t.Errorf("Data.SourceMask() = %v, want %v", got, tt.mask)
			}
		})
	}
}
