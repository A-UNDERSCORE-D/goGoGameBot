package interfaces

import (
	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"
)

type Bot interface {
	Error(error)
	IRCMessager
	IRCHooker
	CommandHooker
}

type IRCMessager interface {
	SendPrivmsg(target, message string)
	SendNotice(target, message string)
	WriteString(message string) error
	WriteIRCLine(line ircmsg.IrcMessage) error
}

type IRCHooker interface {
	HookPrivmsg(f func(source, target, message string, originalLine ircmsg.IrcMessage, bot Bot))
	HookRaw(command string, callback func(ircmsg.IrcMessage, Bot), priority int)
}

type CmdFunc func(fromIRC bool, args []string, source ircutils.UserHost, target string)
type CommandHooker interface {
	HookCommand(name string, adminRequired int, help string, callback CmdFunc) error
	HookSubCommand(rootCommand, name string, adminRequired int, help string, callback CmdFunc) error
	UnhookCommand(name string) error
	UnhookSubCommand(rootName, name string) error
}
