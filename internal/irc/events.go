package irc

import (
	"strings"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// RawEvent represents an incoming raw IRC Line that needs to be handled
type RawEvent struct {
	// TODO: have this just embed the Line?
	event.BaseEvent
	Line ircmsg.IrcMessage
	Time time.Time
}

// CommandIs returns whether or not the command on the Line contained in the RawEvent matches any of the passed command
// names
func (r *RawEvent) CommandIs(names ...string) bool {
	for _, n := range names {
		if n == r.Line.Command {
			return true
		}
	}
	return false
}

// NewRawEvent creates a RawEvent with the given name and Line
func NewRawEvent(name string, line ircmsg.IrcMessage, tme time.Time) *RawEvent {
	return &RawEvent{event.BaseEvent{Name_: strings.ToUpper(name)}, line, tme}
}

// MessageEvent represents an IRC authUser message, both NOTICE and PRIVMSGs
type MessageEvent struct {
	*RawEvent
	IsNotice bool
	Source   ircutils.UserHost
	Channel  string
	Message  string
}

// NewMessageEvent creates a MessageEvent with the given data.
func NewMessageEvent(name string, line ircmsg.IrcMessage, tme time.Time) *MessageEvent {
	return &MessageEvent{
		NewRawEvent(name, line, tme),
		line.Command == "NOTICE",
		ircutils.ParseUserhost(line.Prefix),
		util.IdxOrEmpty(line.Params, 0),
		util.IdxOrEmpty(line.Params, 1),
	}
}

// JoinEvent represents an IRC channel JOIN
type JoinEvent struct {
	*RawEvent
	Source  ircutils.UserHost
	Channel string
}

// NewJoinEvent creates a JoinEvent with the given data
func NewJoinEvent(name string, line ircmsg.IrcMessage, tme time.Time) *JoinEvent {
	return &JoinEvent{
		NewRawEvent(name, line, tme),
		ircutils.ParseUserhost(line.Prefix),
		util.IdxOrEmpty(line.Params, 0),
	}
}

// PartEvent represents an IRC channel PART
type PartEvent struct {
	*JoinEvent
	Message string
}

// NewPartEvent creates a PartEvent with the given data
func NewPartEvent(name string, line ircmsg.IrcMessage, tme time.Time) *PartEvent {
	return &PartEvent{
		JoinEvent: NewJoinEvent(name, line, tme),
		Message:   util.IdxOrEmpty(line.Params, 1),
	}
}

// QuitEvent represents an IRC QUIT
type QuitEvent struct {
	*RawEvent
	Source  ircutils.UserHost
	Message string
}

// NewQuitEvent creates a QuitEvent from the given data
func NewQuitEvent(name string, line ircmsg.IrcMessage, tme time.Time) *QuitEvent {
	return &QuitEvent{
		RawEvent: NewRawEvent(name, line, tme),
		Source:   ircutils.ParseUserhost(line.Prefix),
		Message:  util.IdxOrEmpty(line.Params, 0),
	}
}

// NickEvent represents an IRC NICK command
type NickEvent struct {
	*RawEvent
	Source  ircutils.UserHost
	NewNick string
}

// NewNickEvent creates a NickEvent from the given data
func NewNickEvent(name string, line ircmsg.IrcMessage, tme time.Time) *NickEvent {
	return &NickEvent{
		RawEvent: NewRawEvent(name, line, tme),
		Source:   ircutils.ParseUserhost(line.Prefix),
		NewNick:  util.IdxOrEmpty(line.Params, 0),
	}
}

// KickEvent represents a channel KICK
type KickEvent struct {
	*RawEvent
	Source     ircutils.UserHost
	Channel    string
	KickedNick string
	Message    string
}

// NewKickEvent creates a KickEvent from the given data
func NewKickEvent(name string, line ircmsg.IrcMessage, tme time.Time) *KickEvent {
	return &KickEvent{
		RawEvent:   NewRawEvent(name, line, tme),
		Source:     ircutils.ParseUserhost(line.Prefix),
		Channel:    util.IdxOrEmpty(line.Params, 0),
		KickedNick: util.IdxOrEmpty(line.Params, 1),
		Message:    util.IdxOrEmpty(line.Params, 2),
	}
}
