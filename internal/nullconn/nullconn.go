// Package nullconn holds a null connection that satisfies interfaces.Bot
package nullconn

import (
	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
)

// New creates a new NullConn for use with a bot
func New(l *log.Logger) *NullConn {
	return &NullConn{l, make(chan error)}
}

// NullConn is an implementation of interfaces.Bot that is one giant no-op (with logging)
// Its good for testing or simply to use gggb as a management engine for games
// without the chat portions
type NullConn struct {
	log        *log.Logger
	disconChan chan error
}

// Connect connects to the service. It MUST NOT block after negotiation with the service is complete
func (n *NullConn) Connect() error { n.log.Infof("connect requested"); return nil }

// Disconnect disconnects the bot from the service accepts a message for a reason, and should make Run return
func (n *NullConn) Disconnect(msg string) {
	n.log.Infof("Disconnect requested with message: %s", msg)
	n.disconChan <- nil
}

// Run runs the bot, it should Connect() to the service and MUST block until connection is lost
func (n *NullConn) Run() error { n.log.Info("Run requested"); return <-n.disconChan }

// SendAdminMessage sends the given message to the administrator
func (n *NullConn) SendAdminMessage(msg string) { n.log.Infof("Admin message: %s", msg) }

// SendMessage sends a message to the given target
func (n *NullConn) SendMessage(target, msg string) { n.log.Infof("message to %s: %s", target, msg) }

// SendNotice sends a message that should not be responded to to the given target
func (n *NullConn) SendNotice(target, msg string) { n.log.Infof("Notice to %s: %s", target, msg) }

// HookMessage hooks on messages to a channel
func (n *NullConn) HookMessage(_ func(source, channel, message string, isAction bool)) {}

// HookPrivateMessage hooks on messages to us directly
func (n *NullConn) HookPrivateMessage(_ func(source, channel, message string)) {}

// HookJoin hooks on users joining a channel
func (n *NullConn) HookJoin(_ func(source, channel string)) {}

// HookPart hooks on users leaving a channel
func (n *NullConn) HookPart(_ func(source, channel, message string)) {}

// HookQuit hooks on users disconnecting
// (for services that differentiate it from the above)
func (n *NullConn) HookQuit(_ func(source, message string)) {}

// HookKick hooks on a user being kicked from a channel
func (n *NullConn) HookKick(_ func(source, channel, target, message string)) {}

// HookNick hoops on a user changing their nickname
func (n *NullConn) HookNick(_ func(source, newNick string)) {}

// AdminLevel returns the permission level that a given source has. It should return
// zero for sources with no permissions
func (n *NullConn) AdminLevel(source string) int { return 0 }

// JoinChannel joins the given channel
func (n *NullConn) JoinChannel(name string) { n.log.Infof("join channel requested: %s", name) }

// Reload reloads the Bot using the given (string) config
func (n *NullConn) Reload(conf interfaces.Unmarshaler) error {
	n.log.Infof("reload requested with conf %#v", conf)
	return nil
}

// StaticCommandPrefixes returns the bot's current command prefixes
func (n *NullConn) StaticCommandPrefixes() []string { return nil }

// IsCommandPrefix returns the given string and nothing else, as we dont implement it
func (n *NullConn) IsCommandPrefix(l string) (string, bool) { return l, false }

// HumanReadableSource converts the given message source to one that is human readable
func (n *NullConn) HumanReadableSource(source string) string { return source }

// Status returns the status of the null config
func (n *NullConn) Status() string { return "I am a meat popsicle" }

// SendRaw does exactly nothing.
func (n *NullConn) SendRaw(string) {}
