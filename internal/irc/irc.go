package irc

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/keepalive"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/mutexTypes"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// Admin holds a mask level pair, for use in commands
type Admin struct {
	Mask  string `xml:"mask,attr"`
	Level int    `xml:"level,attr"`
}

// Conf holds the configuration for an IRC instance
type Conf struct {
	DontVerifyCerts bool   `xml:"dont_verify_certs,attr"`
	SSL             bool   `xml:"ssl,attr"`
	CmdPfx          string `xml:"command_prefix,attr"`

	Host          string   `xml:"host"`
	Port          string   `xml:"port"`
	HostPasswd    string   `xml:"host_password"`
	Admins        []Admin  `xml:"admin"`
	AdminChannels []string `xml:"admin_channel"`

	Nick  string `xml:"nick"`
	Ident string `xml:"ident"`
	Gecos string `xml:"gecos"`

	// TODO: Cert based auth? (via SASL)
	Authenticate bool   `xml:"authenticate"`
	SASL         bool   `xml:"use_sasl"`
	AuthUser     string `xml:"auth_user"`
	AuthPasswd   string `xml:"auth_password"`
}

// IRC Represents a connection to an IRC server
type IRC struct {
	*Conf
	channels          mutexTypes.StringSlice
	Connected         mutexTypes.Bool
	StopRequested     mutexTypes.Bool
	socket            net.Conn
	socketDoneChan    chan struct{} // Sentinel for when the socket dies
	lag               mutexTypes.Duration
	lastPong          mutexTypes.Time
	log               *log.Logger
	RawEvents         *event.Manager
	ParsedEvents      *event.Manager
	capabilityManager *capabilityManager
}

// New creates a new IRC instance ready for use
func New(conf string, logger *log.Logger) (*IRC, error) {
	out := &IRC{
		log:            logger,
		RawEvents:      new(event.Manager),
		ParsedEvents:   new(event.Manager),
		socketDoneChan: make(chan struct{}),
	}

	if err := out.Reload(conf); err != nil {
		return nil, err
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
	i.RawEvents.Attach("NOTICE", i.dispatchMessage, event.PriHighest)
	i.RawEvents.Attach("JOIN", i.dispatchJoin, event.PriHighest)
	i.RawEvents.Attach("PART", i.dispatchPart, event.PriHighest)
	i.RawEvents.Attach("QUIT", i.dispatchQuit, event.PriHighest)
	i.RawEvents.Attach("KICK", i.dispatchKick, event.PriHighest)
	i.RawEvents.Attach("NICK", i.dispatchNick, event.PriHighest)
	i.RawEvents.Attach("PONG", i.pongHandler, event.PriHighest)
}

// LineHandler is a function that is called on every raw Line
type LineHandler func(message *ircmsg.IrcMessage, irc *IRC)

// ErrNotConnected returned from Write when the IRC instance is not connected to a server
var ErrNotConnected = errors.New("cannot send a message when not connected")

func (i *IRC) write(toSend []byte) (int, error) {
	if !i.Connected.Get() {
		return 0, ErrNotConnected
	}
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
	i.Connected.Set(true)
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

	i.RawEvents.WaitFor("001")

	for _, name := range i.channels.Get() {
		_, _ = i.writeLine("JOIN", name)
	}
	go i.pingLoop()

	return nil
}

// Disconnect disconnects the bot from IRC either with the given message, or the message "Disconnecting" when none is passed
func (i *IRC) Disconnect(msg string) {
	if msg != "" {
		_, _ = i.writeLine("QUIT", msg)
	} else {
		_, _ = i.writeLine("QUIT", "Disconnecting")
	}
	i.StopRequested.Set(true)
	go func() {
		time.Sleep(time.Millisecond * 30)
		if i.Connected.Get() {
			i.socket.Close()
		}
	}()
}

// Run connects the bot and blocks until it disconnects
func (i *IRC) Run() error {
	// ensure that if the socket chan was marked as closed, we clean it up
	select {
	case <-i.socketDoneChan:
	default:
	}

	if err := i.Connect(); err != nil {
		return err
	}
	defer i.Connected.Set(false)
	select {
	case e := <-i.RawEvents.WaitForChan("ERROR"):
		if !i.StopRequested.Get() {
			return fmt.Errorf("IRC server sent us an ERROR line: %s", event2RawEvent(e).Line)
		}
	case <-i.socketDoneChan:
		return fmt.Errorf("IRC socket closed")
	}
	return nil
}

func (i *IRC) pingLoop() {
	for i.Connected.Get() {
		time.Sleep(time.Second * 5)
		msg := fmt.Sprintf("%s %s", time.Now().Format(time.RFC3339Nano), keepalive.Next())
		if _, err := i.writeLine("PING", msg); err != nil {
			// Something broke. No idea what. Bail out the entire bot
			i.socket.Close()
		}
		i.checkLag()
	}
}

func (i *IRC) pongHandler(e event.Event) {
	rawEvent := event2RawEvent(e)
	if rawEvent == nil {
		i.log.Warnf("Got an invalid PONG")
		return
	}
	ts := strings.SplitN(rawEvent.Line.Params[len(rawEvent.Line.Params)-1], " ", 2)[0]
	thyme, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		i.log.Warnf("could not parse time for PONG: %s", err)
		return
	}

	i.lag.Set(time.Since(thyme))
	i.lastPong.Set(time.Now())
	i.checkLag()
}

func (i *IRC) checkLag() {
	lp := i.lastPong.Get()
	if !lp.IsZero() && time.Since(lp) > time.Second*30 {
		i.Disconnect(fmt.Sprintf("No ping response in %s", time.Since(lp)))
	}
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
				i.log.Warnf("IRC.readloop(): could not create ping: %s", err)
			}

		}
		i.checkLag()
		i.handleLine(line)
	}
	i.log.Info("IRC socket closed")
	i.socketDoneChan <- struct{}{}
}

func (i *IRC) handleLine(line ircmsg.IrcMessage) {
	t := time.Now()
	if i.capabilityManager.capEnabled("server-time") && line.HasTag("time") {
		_, timeFromServer := line.GetTag("time")
		serverTime, err := time.Parse(time.RFC3339, timeFromServer)
		if err != nil {
			i.log.Warnf("server offered server-time %q which does not fit RFC 3339 format: %s", timeFromServer, err)
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

func nickOrOriginal(toParse string) string {
	parsed := ircutils.ParseUserhost(toParse)
	if parsed.Nick != "" {
		return parsed.Nick
	}
	return toParse
}

// SendMessage sends a message to the given target
func (i *IRC) SendMessage(target, message string) {
	msgs := strings.Split(message, "\n")
	for _, m := range msgs {
		if _, err := i.writeLine("PRIVMSG", nickOrOriginal(target), ircTransformer.Transform(m)); err != nil {
			i.log.Warnf("could not send message %q to target %q: %s", m, target, err)
		}
	}
}

// SendNotice sends a notice to the given notice
func (i *IRC) SendNotice(target, message string) {
	msgs := strings.Split(message, "\n")
	for _, m := range msgs {
		if _, err := i.writeLine("NOTICE", nickOrOriginal(target), ircTransformer.Transform(m)); err != nil {
			i.log.Warnf("could not send notice %q to target %q: %s", m, target, err)
		}
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

// JoinChannel joins the bot to the named channel and adds it to the channel list for later autojoins
func (i *IRC) JoinChannel(name string) {
	if i.Connected.Get() {
		i.writeLine("JOIN", name)
	}

	i.channels.Set(append(i.channels.Get(), name))
}

func (i *IRC) String() string {
	return fmt.Sprintf(
		"IRC conn; Host: %s, Port: %s, Conencted: %t, Lag: %dms",
		i.Host,
		i.Port,
		i.Connected.Get(),
		i.lag.Get().Milliseconds(),
	)
}

// Reload parses and reloads the config on the IRC instance
func (i *IRC) Reload(conf string) error {
	newConf := new(Conf)
	if err := xml.Unmarshal([]byte(conf), newConf); err != nil {
		return fmt.Errorf("could not parse config: %s", err)
	}
	i.Conf = newConf
	return nil
}

// CommandPrefixes returns the valid command prefixes for the IRC instance.
// Specifically, the configured one, and the set nick followed by a colon
func (i *IRC) CommandPrefixes() []string {
	return []string{i.CmdPfx, i.Nick + ": "}
}

// HumanReadableSource takes an IRC userhost and returns just the nick
func (i *IRC) HumanReadableSource(source string) string {
	if out := ircutils.ParseUserhost(source).Nick; out != "" {
		return out
	}

	return source
}

func (i *IRC) Status() string {
	return fmt.Sprintf("IRC: Connected: %t Lag: %s", i.Connected.Get(), i.lag.Get())
}
