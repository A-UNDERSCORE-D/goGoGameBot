package util

import "github.com/goshuirc/irc-go/ircmsg"

func MakeSimpleIRCLine(command string, args ...string) ircmsg.IrcMessage {
    return ircmsg.MakeMessage(nil, "", command, args...)
}
