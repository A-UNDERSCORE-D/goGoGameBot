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
	authUser     string
	password     string

	m                 sync.RWMutex
	socket            net.Conn
	log               *log.Logger
	RawEvents         *event.Manager
	ParsedEvents      *event.Manager
	capabilityManager *capabilityManager
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
		authUser:     conf.NSAuth.Nick,
		password:     conf.NSAuth.Password,
		log:          logger,
		RawEvents:    new(event.Manager),
		ParsedEvents: new(event.Manager),
	}
	out.setupParsers()
	out.capabilityManager = newCapabilityManager(out)
	out.capabilityManager.supportCap("userhost-in-names")
	out.capabilityManager.supportCap("server-time")

	out.capabilityManager.supportCap("sasl")
	out.capabilityManager.CapEvents.Attach("sasl", out.authenticateWithSasl, event.PriNorm)

	return out
}

func (i *IRC) setupParsers() {
	i.RawEvents.Attach("RAW_PRIVMSG", i.dispatchMessage, event.PriHighest)
}

func (i *IRC) Run() {
	if err := i.connect(); err != nil {
		panic(err)
	}

	i.RawEvents.WaitFor("ERROR")
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
	l := util.MakeSimpleIRCLine(command, args...)
	lBytes, err := l.LineBytes()
	if err != nil {
		return -1, err
	}
	return i.write(lBytes)
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
	go i.readLoop()

	if i.hostPassword != "" {
		i.writeLine("PASS", i.hostPassword)
	}
	i.capabilityManager.negotiateCaps()
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

		i.handleLine(line)
	}
}

func (i *IRC) handleLine(line ircmsg.IrcMessage) {
	t := time.Now()
	if i.capabilityManager.capEnabled("server-time") && line.HasTag("time") {
		_, timeFromServer := line.GetTag("time")
		serverTime, err := time.Parse(time.RFC3339, timeFromServer)
		if err != nil {
			i.log.Warnf("server offered server-time %q which does not fit RFC 3339 format")
		} else {
			t = serverTime
		}
	}

	i.RawEvents.Dispatch(NewRawEvent(line.Command, line, t))
	i.RawEvents.Dispatch(NewRawEvent("*", line, t))
}

func (i *IRC) authenticateWithSasl(e event.Event) {
	const authenticate = "AUTHENTICATE"
	capab := e.(*capEvent).cap
	_ = capab
	capChan := make(chan event.Event, 1)
	exiting := make(chan struct{})
	id := i.RawEvents.AttachMany(func(e event.Event) {
		select {
		case capChan <- e:
		case _, _ = <-exiting:
		}
	}, event.PriNorm,
		authenticate,
		util.RPL_LOGGEDIN,
		util.RPL_LOGGEDOUT,
		util.RPL_NICKLOCKED,
		util.RPL_SASLSUCCESS,
		util.RPL_SASLFAIL,
		util.RPL_SASLTOOLONG,
		util.RPL_SASLABORTED,
		util.RPL_SASLALREADY,
		util.RPL_SASLMECHS,
	)
	defer close(exiting)
	defer i.RawEvents.Detach(id)

	if _, err := i.writeLine(authenticate, "PLAIN"); err != nil {
		i.log.Warn("authenticateWithSasl(): could not send SASL authentication request. Aborting SASL")
		i.sasl = false
		return
	}

	for e := range capChan {
		raw := event2RawEvent(e)
		if raw == nil {
			i.log.Warn("authenticateWithSasl(): got an unexpected event over the event channel: ", e)
			continue
		}
		switch raw.Line.Command {
		case authenticate:
			if raw.Line.Params[0] == "+" {
				_, err := i.writeLine(authenticate, util.GenerateSASLString(i.nick, i.authUser, i.password))
				if err != nil {
					i.log.Warn("authenticateWithSasl(): could not send SASL authentication. Aborting")
					i.sasl = false
					return
				}
			}

		case util.RPL_NICKLOCKED, util.RPL_SASLFAIL, util.RPL_SASLTOOLONG, util.RPL_SASLABORTED,
			util.RPL_SASLALREADY, util.RPL_SASLMECHS:
			i.log.Warn("authenticateWithSasl(): SASL negotiation failed. Aborting")
			i.sasl = false
			return
		case util.RPL_LOGGEDIN, util.RPL_SASLSUCCESS:
			// it worked \o/
			return
		default:
			i.log.Warn("authenticateWithSasl(): got an unexpected command over the event channel: ", raw)
		}
	}

}
