package irc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// Admin holds a mask level pair, for use in commands
type Admin struct {
	Mask  string `xml:"mask"`
	Level int    `xml:"level"`
}

// Conf holds the configuration for an IRC instance
type Conf struct {
	Host            string `xml:"host"`
	Port            string `xml:"port"`
	HostPasswd      string `xml:"host_password"`
	SSL             bool   `xml:"ssl,attr"`
	DontVerifyCerts bool   `xml:"dont_verify_certs,attr"`
	Admins          []Admin
	AdminChannels   []string

	Nick  string `xml:"nick"`
	Ident string `xml:"ident"`
	Gecos string `xml:"gecos"`

	// TODO: Cert based auth? (via SASL)
	Authenticate bool   `xml:"authenticate,attr"`
	SASL         bool   `xml:"sasl,attr"`
	AuthUser     string `xml:"auth_user"`
	AuthPasswd   string `xml:"auth_password"`
}

// IRC Represents a connection to an IRC server
type IRC struct {
	Conf
	m                 sync.RWMutex
	socket            net.Conn
	log               *log.Logger
	RawEvents         *event.Manager
	ParsedEvents      *event.Manager
	capabilityManager *capabilityManager
}

// TODO: rename config.BotConfig config.IRCConfig

// New creates a new IRC instance ready for use
func New(conf string, logger *log.Logger) (*IRC, error) {
	c := new(Conf)
	if err := xml.Unmarshal([]byte(conf), c); err != nil {
		return nil, err
	}

	out := &IRC{
		log:          logger,
		RawEvents:    new(event.Manager),
		ParsedEvents: new(event.Manager),
	}

	if out.DontVerifyCerts {
		out.log.Warn("IRC instance created without certificate verification. This is susceptible to MITM attacks")
	}

	out.setupParsers()
	out.capabilityManager = newCapabilityManager(out)
	out.capabilityManager.supportCap("userhost-in-names")
	out.capabilityManager.supportCap("server-time")

	if out.SSL {
		out.capabilityManager.supportCap("sasl")
		out.capabilityManager.CapEvents.Attach("sasl", out.authenticateWithSasl, event.PriNorm)
	} else if out.SASL {
		out.SASL = false
		out.log.Warn("SASL disabled as the connection is not SSL")
	}

	return out, nil
}

func (i *IRC) setupParsers() {
	i.RawEvents.Attach("PRIVMSG", i.dispatchMessage, event.PriHighest)
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

// Connect connects to IRC and does the required negotiation for registering on the network and any capabilities
// that have been requested
func (i *IRC) Connect() error {
	target := net.JoinHostPort(i.Host, i.Port)
	var s net.Conn
	var err error
	if i.SSL {
		s, err = tls.Dial("tcp", target, &tls.Config{InsecureSkipVerify: i.DontVerifyCerts})
	} else {
		s, err = net.Dial("tcp", target)
	}

	if err != nil {
		return fmt.Errorf("IRC.Connect(): could not open socket: %s", err)
	}
	i.socket = s
	go i.readLoop()

	if i.HostPasswd != "" {
		if _, err := i.writeLine("PASS", i.HostPasswd); err != nil {
			return err
		}
	}

	i.capabilityManager.negotiateCaps()
	if _, err := i.writeLine("USER", i.Ident, "*", "*", i.Gecos); err != nil {
		return err
	}
	if _, err := i.writeLine("NICK", i.Nick); err != nil {
		return err
	}

	if !i.SASL && i.Authenticate {
		i.RawEvents.AttachOneShot("001", func(e event.Event) {
			raw := event2RawEvent(e)
			if raw == nil {
				i.log.Warn("unexpected event type in event handler: ", e)
				return
			}
			i.SendMessage("NickServ", fmt.Sprintf("IDENTIFY %s %s", i.AuthUser, i.AuthPasswd))
		}, event.PriHighest)
	}

	return nil
}

func (i *IRC) Disconnect(msg string) {
	if msg != "" {
		i.writeLine("QUIT", msg)
	} else {
		i.writeLine("QUIT", "Disconnecting")
	}
}

// Run connects the bot and blocks until it disconnects
func (i *IRC) Run() error {
	if err := i.Connect(); err != nil {
		return err
	}
	i.RawEvents.WaitFor("ERROR")
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
		i.SASL = false
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
				_, err := i.writeLine(authenticate, util.GenerateSASLString(i.Nick, i.AuthUser, i.AuthPasswd))
				if err != nil {
					i.log.Warn("authenticateWithSasl(): could not send SASL authentication. Aborting")
					i.SASL = false
					return
				}
			}

		case util.RPL_NICKLOCKED, util.RPL_SASLFAIL, util.RPL_SASLTOOLONG, util.RPL_SASLABORTED,
			util.RPL_SASLALREADY, util.RPL_SASLMECHS:
			i.log.Warn("authenticateWithSasl(): SASL negotiation failed. Aborting")
			i.SASL = false
			return
		case util.RPL_LOGGEDIN, util.RPL_SASLSUCCESS:
			// it worked \o/
			return
		default:
			i.log.Warn("authenticateWithSasl(): got an unexpected command over the event channel: ", raw)
		}
	}

}

// SendMessage sends a message to the given target
func (i *IRC) SendMessage(target, message string) {
	if _, err := i.writeLine("PRIVMSG", ircutils.ParseUserhost(target).Nick, message); err != nil {
		i.log.Warnf("could not send message %q to target %q: %s", message, target, err)
	}
}

// SendNotice sends a notice to the given notice
func (i *IRC) SendNotice(target, message string) {
	if _, err := i.writeLine("NOTICE", ircutils.ParseUserhost(target).Nick, message); err != nil {
		i.log.Warnf("could not send notice %q to target %q: %s", message, target, err)
	}
}

// AdminLevel returns what admin level the given mask has, 0 means no admin access
func (i *IRC) AdminLevel(source string) int {
	for _, a := range i.Admins {
		if util.GlobToRegexp(a.Mask).MatchString(source) {
			return a.Level
		}
	}
	return 0
}

// SendAdminMessage sends the given message to all AdminChannels defined on the bot
func (i *IRC) SendAdminMessage(msg string) {
	for _, c := range i.AdminChannels {
		i.SendMessage(c, msg)
	}
}
