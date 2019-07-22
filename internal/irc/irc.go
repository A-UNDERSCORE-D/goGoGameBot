package irc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// IRC Represents a connection to an IRC server
type IRC struct {
	host         string
	port         string
	hostPassword string
	ssl          bool
	ignoreCert   bool

	nick  string
	ident string
	gecos string

	// TODO: Cert based auth? (via SASL)
	authenticate bool
	sasl         bool
	user         string
	password     string

	m            sync.RWMutex
	socket       net.Conn
	log          *log.Logger
	RawEvents    *event.Manager
	ParsedEvents *event.Manager
}

// TODO: rename config.BotConfig config.IRCConfig

// New creates a new IRC instance ready for use
func New(conf config.BotConfig, logger *log.Logger) *IRC {
	/*
		TODO:
			if conf.DontVerifyCerts {
				logger.Warn("IRC instance created without certificate verification. This is susceptible to MITM attacks")
			}
	*/

	out := &IRC{
		host:         conf.Host,
		port:         conf.Port,
		hostPassword: "", // TODO
		ssl:          conf.SSL,
		ignoreCert:   false, // TODO
		nick:         conf.Nick,
		ident:        conf.Ident,
		gecos:        conf.Gecos,
		authenticate: !(conf.NSAuth.Nick == "" || conf.NSAuth.Password == ""),
		sasl:         conf.NSAuth.SASL,
		user:         conf.NSAuth.Nick,
		password:     conf.NSAuth.Password,
		log:          logger,
		RawEvents:    new(event.Manager),
		ParsedEvents: new(event.Manager),
	}
	out.setupParsers()

	return out
}

func (i *IRC) setupParsers() {
	i.RawEvents.Attach("RAW_PRIVMSG", i.dispatchMessage, event.PriHighest)
	i.RawEvents.Attach("RAW_NOTICE", i.dispatchMessage, event.PriHighest)
}

// LineHandler is a function that is called on every raw Line
type LineHandler func(message *ircmsg.IrcMessage, irc *IRC)

func (i *IRC) write(toSend []byte) (int, error) {
	if !bytes.HasSuffix(toSend, []byte{'\r', '\n'}) {
		toSend = append(toSend, '\r', '\n')
	}
	i.log.Debug("<< ", string(toSend))
	return i.socket.Write(toSend)
}

func (i *IRC) writeLine(command string, args ...string) (int, error) {
	l, err := util.MakeSimpleIRCLine(command, args...).LineBytes()
	if err != nil {
		return -1, err
	}
	return i.write(l)
}

func (i *IRC) connect() error {
	target := net.JoinHostPort(i.host, i.port)
	var s net.Conn
	var err error
	if i.ssl {
		s, err = tls.Dial("tcp", target, &tls.Config{InsecureSkipVerify: i.ignoreCert})
	} else {
		s, err = net.Dial("tcp", target)
	}

	if err != nil {
		return fmt.Errorf("IRC.Connect(): could not open socket: %s", err)
	}
	i.socket = s

	if i.hostPassword != "" {
		i.writeLine("PASS", i.hostPassword)
	}

	i.writeLine("USER", i.ident, "*", "*", i.gecos)
	i.writeLine("NICK", i.nick)
	return nil
}

func (i *IRC) readLoop() {
	s := bufio.NewScanner(i.socket)
	for s.Scan() {
		str := s.Text()
		i.log.Debug(">> ", str)
		line, err := ircmsg.ParseLine(str)
		if err != nil {
			i.log.Warnf("IRC.readLoop(): Discarding invalid Line %q: %s", str, err)
			continue
		}

		if line.Command == "PING" {
			if _, err := i.writeLine("PONG", line.Params...); err != nil {
				panic(fmt.Errorf("IRC.readloop(): could not create ping. This is a bug: %s", err))
			}

		}

	}
}

func (i *IRC) handleLine(line *ircmsg.IrcMessage) error {
	t := time.Now() // TODO: When server time is supported, use that
	i.RawEvents.Dispatch(NewRawEvent(line.Command, line, t))
	i.RawEvents.Dispatch(NewRawEvent("*", line, t))
	return nil
}
