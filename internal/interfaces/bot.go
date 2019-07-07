package interfaces

import (
	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"
)

// Bot represents an IRC bot
type Bot interface {
	Error(error)
	IRCMessager
	IRCHooker
	CommandHooker
}

// IRCMessager represents a type that can send messages to an IRC network
type IRCMessager interface {
	SendPrivmsg(target, message string)
	SendNotice(target, message string)
	WriteString(message string) error
	WriteIRCLine(line ircmsg.IrcMessage) error
}

// IRCHooker provides methods to hook callback functions onto IRC commands, with a helper specifically for IRC PRIVMSGs
type IRCHooker interface {
	HookPrivmsg(f func(source, target, message string, originalLine ircmsg.IrcMessage, bot Bot))
	HookRaw(command string, callback func(ircmsg.IrcMessage, Bot), priority int)
}

// CommandResponder provides helper methods for responding to command calls with Messages, and Notices
type CommandResponder interface {
	ReturnNotice(msg string)
	ReturnMessage(msg string)
}

// CmdFunc is a function used in a command callback
type CmdFunc func(fromIRC bool, args []string, source ircutils.UserHost, target string, responder CommandResponder)

// CommandHooker provides methods to hook and unhook callbacks onto commands
type CommandHooker interface {
	HookCommand(name string, adminRequired int, help string, callback CmdFunc) error
	HookSubCommand(rootCommand, name string, adminRequired int, help string, callback CmdFunc) error
	UnhookCommand(name string) error
	UnhookSubCommand(rootName, name string) error
}
