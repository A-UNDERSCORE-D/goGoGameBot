package irc

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/goshuirc/irc-go/ircmsg"
	"github.com/goshuirc/irc-go/ircutils"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
	"awesome-dragon.science/go/goGoGameBot/pkg/event"
	"awesome-dragon.science/go/goGoGameBot/pkg/keepalive"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
	"awesome-dragon.science/go/goGoGameBot/pkg/mutexTypes"
	"awesome-dragon.science/go/goGoGameBot/pkg/util"
)

// Admin holds a mask level pair, for use in commands
type Admin struct {
	Mask  string `xml:"mask,attr"`
	Level int    `xml:"level,attr"`
}

// Conf holds the configuration for an IRC instance
type Conf struct {
	VerifyCerts bool   `toml:"verify_certs" default:"true" comment:"verify TLS certs when connecting (default: true)"`
	SSL         bool   `toml:"ssl" default:"true" comment:"Use SSL/TLS (really TLS) for connection (default: true)"`
	CmdPfx      string `toml:"command_prefix" default:"~" comment:"Command prefix to respond to (default: '~')"`

	Host          string   `toml:"host"`
	Port          string   `toml:"port"`
	HostPasswd    string   `toml:"host_password"`
	Admins        []Admin  `toml:"admins"`
	AdminChannels []string `toml:"admin_channels"`

	Nick  string `toml:"nick"`
	Ident string `toml:"ident"`
	Gecos string `toml:"gecos"`

	// TODO: Cert based auth? (via SASL)
	Authenticate bool   `toml:"authenticate" comment:"Should we authenticate with the IRC network"`
	SASL         bool   `toml:"sasl" comment:"Should authentication use SASL for negotiation (this is faster and more secure)"` //nolint:lll // Cant be made shorter
	AuthUser     string `toml:"auth_user" comment:"User account to authenticate for"`
	AuthPasswd   string `toml:"auth_passwd" comment:"Password for account authentication"`

	SuppressMOTD bool `toml:"suppress_motd" comment:"Suppress logging of IRC MOTD messages being logged"`
	SuppressPing bool `toml:"suppress_ping" comment:"Suppress logging of internal PING messages being logged"`
}

// IRC Represents a connection to an IRC server
type IRC struct {
	*Conf
	runtimeNick       string
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
func New(conf tomlconf.ConfigHolder, logger *log.Logger) (*IRC, error) {
	out := &IRC{
		log:            logger,
		RawEvents:      new(event.Manager),
		ParsedEvents:   new(event.Manager),
		socketDoneChan: make(chan struct{}),
	}

	if err := out.Reload(conf.RealConf); err != nil {
		return nil, err
	}

	if !out.VerifyCerts {
		out.log.Warn("IRC instance created without certificate verification. This is susceptible to MITM attacks")
	}

	out.setupParsers()

	return out, nil
}

func (i *IRC) setupCapManager() {
	i.capabilityManager = newCapabilityManager(i)
	i.capabilityManager.supportCap("userhost-in-names")
	i.capabilityManager.supportCap("server-time")

	if i.SSL && i.SASL {
		i.capabilityManager.supportCap("sasl")
		i.capabilityManager.CapEvents.Attach("sasl", i.authenticateWithSasl, event.PriNorm)
	} else if i.SASL {
		i.SASL = false
		i.log.Warn("SASL disabled as the connection is not SSL")
	}
}

func (i *IRC) setupParsers() {
	// Dispatchers
	i.RawEvents.Attach("PRIVMSG", i.dispatchMessage, event.PriHighest)
	i.RawEvents.Attach("NOTICE", i.dispatchMessage, event.PriHighest)
	i.RawEvents.Attach("JOIN", i.dispatchJoin, event.PriHighest)
	i.RawEvents.Attach("PART", i.dispatchPart, event.PriHighest)
	i.RawEvents.Attach("QUIT", i.dispatchQuit, event.PriHighest)
	i.RawEvents.Attach("KICK", i.dispatchKick, event.PriHighest)
	i.RawEvents.Attach("NICK", i.dispatchNick, event.PriHighest)
	i.RawEvents.Attach("PONG", i.pongHandler, event.PriHighest)

	// internal handlers
	i.RawEvents.Attach("433", i.handleNickInUse, event.PriHighest)
	i.HookNick(i.onNick)
}

// LineHandler is a function that is called on every raw Line
type LineHandler func(message *ircmsg.IrcMessage, irc *IRC)

// ErrNotConnected returned from Write when the IRC instance is not connected to a server
var ErrNotConnected = errors.New("cannot send a message when not connected")

func (i *IRC) write(toSend []byte, logMsg bool) (int, error) {
	if !i.Connected.Get() {
		return 0, ErrNotConnected
	}

	out := make([]byte, len(toSend))
	copy(out, toSend)

	if !bytes.HasSuffix(out, []byte{'\r', '\n'}) {
		out = append(out, '\r', '\n')
	}

	if logMsg {
		i.log.Debug("<< ", string(out[:len(out)-2]))
	}

	return i.socket.Write(out)
}

//nolint:unparam // Its to mimic the Write interface
func (i *IRC) writeLine(command string, args ...string) (int, error) {
	l := util.MakeSimpleIRCLine(command, args...)
	lBytes, err := l.LineBytes()

	if err != nil {
		return -1, err
	}

	return i.write(lBytes, !(command == "PING" && i.SuppressPing))
}

const maxDialTimeout = time.Second * 10

// Connect connects to IRC and does the required negotiation for registering on the network and any capabilities
// that have been requested
func (i *IRC) Connect() error {
	i.log.Infof("Starting IRC connection to %s:%s SSL: %t", i.Host, i.Port, i.SSL)
	i.setupCapManager()

	target := net.JoinHostPort(i.Host, i.Port)

	var (
		s   net.Conn
		err error
	)

	// TODO: Possibly use a dialer with a context here, to allow cancellation
	dialer := &net.Dialer{Timeout: maxDialTimeout}

	if i.SSL {
		// nolint:gosec // The user explicitly asked for it and we warn about it
		s, err = tls.DialWithDialer(dialer, "tcp", target, &tls.Config{InsecureSkipVerify: !i.VerifyCerts})

		if !i.VerifyCerts {
			i.log.Warnf("**** Not verifying certs. THIS IS INSECURE ****")
		}
	} else {
		s, err = dialer.Dial("tcp", target)
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

	i.capabilityManager.negotiateCaps() // TODO: return an error here

	if _, err := i.writeLine("USER", i.Ident, "*", "*", i.Gecos); err != nil {
		return err
	}

	i.runtimeNick = i.Nick
	if _, err := i.writeLine("NICK", i.Nick); err != nil {
		return err
	}

	select {
	case <-i.RawEvents.WaitForChan("001"):
		if !i.SASL && i.Authenticate {
			i.SendMessage("NickServ", fmt.Sprintf("IDENTIFY %s %s", i.AuthUser, i.AuthPasswd))
		}
	case <-i.socketDoneChan:
		// The socket was closed during
		return errors.New("socket closed while connecting")
	}

	var oErr error

	for _, name := range i.channels.Get() {
		if _, err := i.writeLine("JOIN", name); err != nil {
			oErr = err
			i.log.Warnf("could not write JOIN message: %s", err)
		}
	}

	return oErr
}

// Disconnect disconnects the bot from IRC either with the given message.
// If no message is passed, it defaults to "Disconnecting"
func (i *IRC) Disconnect(msg string) {
	if msg != "" {
		_, _ = i.writeLine("QUIT", msg)
	} else {
		_, _ = i.writeLine("QUIT", "Disconnecting")
	}

	i.StopRequested.Set(true)

	go func() {
		time.Sleep(time.Millisecond * 500)

		if i.Connected.Get() {
			i.log.Warn("disconnect did not happen as expected. forcing a socket close")
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

	pingCtx, cancel := context.WithCancel(context.Background())
	go i.pingLoop(pingCtx)

	defer func() {
		// clean up
		cancel()
		i.lastPong.Set(time.Time{})
		i.lag.Set(time.Duration(0))
		i.Connected.Set(false)
	}()

	select {
	case e := <-i.RawEvents.WaitForChan("ERROR"):
		if !i.StopRequested.Get() {
			return fmt.Errorf("IRC server sent us an ERROR line: %s", strings.Join(event2RawEvent(e).Line.Params, " "))
		}
	case <-i.socketDoneChan:
		return fmt.Errorf("IRC socket closed")
	}

	return nil
}

func (i *IRC) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			i.doPing()
		case <-ctx.Done():
			return
		}
	}
}

func (i *IRC) doPing() {
	msg := fmt.Sprintf("%s %s", time.Now().Format(time.RFC3339Nano), keepalive.Next())

	if _, err := i.writeLine("PING", msg); err != nil {
		i.log.Warnf("could not write our PING message: %s", err)
		// Something broke. No idea what. Bail out the entire bot
		i.socket.Close()
	}

	i.checkLag()
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

var motdNumerics = [...]string{"375", "372", "376"}

func isMotdNumeric(in string) bool {
	for _, v := range motdNumerics {
		if in == v {
			return true
		}
	}

	return false
}

func (i *IRC) shouldSuppressLog(line ircmsg.IrcMessage) bool {
	if isMotdNumeric(line.Command) && i.SuppressMOTD {
		return true
	}

	if line.Command == "PONG" && i.SuppressPing {
		return true
	}

	return false
}

func (i *IRC) readLoop() {
	s := bufio.NewScanner(i.socket)
	for s.Scan() {
		str := s.Text()
		line, err := ircmsg.ParseLine(str)

		if err != nil {
			i.log.Warnf("IRC.readLoop(): Discarding invalid Line %q: %s", str, err)
			continue
		} else if !i.shouldSuppressLog(line) {
			i.log.Debug(">> ", str)
		}

		if line.Command == "PING" {
			if _, err := i.writeLine("PONG", line.Params...); err != nil {
				i.log.Warnf("IRC.readloop(): could not create ping: %s", err)
			}
		}

		i.checkLag()
		i.handleLine(line)
	}

	if err := s.Err(); err != nil {
		i.log.Warnf("Error returned from readLoop scanner: %s", err)
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

	capChan := make(chan event.Event, 1)
	exiting := make(chan struct{})
	id := i.RawEvents.AttachMany(func(e event.Event) {
		select {
		case capChan <- e:
		case <-exiting:
		}
	}, event.PriNorm, authenticate, util.RPL_LOGGEDIN, util.RPL_LOGGEDOUT, util.RPL_NICKLOCKED, util.RPL_SASLSUCCESS,
		util.RPL_SASLFAIL, util.RPL_SASLTOOLONG, util.RPL_SASLABORTED, util.RPL_SASLALREADY, util.RPL_SASLMECHS,
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
		lowerAdmin := strings.ToLower(a.Mask)
		lowerSource := strings.ToLower(source)

		if util.GlobToRegexp(lowerAdmin).MatchString(lowerSource) {
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
		if _, err := i.writeLine("JOIN", name); err != nil {
			i.log.Warn("could not write join command: ", err)
		}
	}

	i.channels.Set(append(i.channels.Get(), name))
}

func (i *IRC) String() string {
	return fmt.Sprintf(
		"IRC[Host[%s], Port:[%s], Connected[%t], Lag[%dms]]",
		i.Host,
		i.Port,
		i.Connected.Get(),
		i.lag.Get().Milliseconds(),
	)
}

// Reload parses and reloads the config on the IRC instance
func (i *IRC) Reload(tree interfaces.Unmarshaler) error {
	newConf := new(Conf)
	if err := tree.Unmarshal(newConf); err != nil {
		return fmt.Errorf("could not unmarshal IRC config: %w", err)
	}

	i.Conf = newConf

	return nil
}

// StaticCommandPrefixes returns the valid static command prefixes. This means
// any prefixes that will never change at runtime (for IRC, none)
func (i *IRC) StaticCommandPrefixes() []string { return []string{} }

// IsCommandPrefix returns whether or not the given string contains any dynamic command prefixes
func (i *IRC) IsCommandPrefix(line string) (string, bool) {
	if strings.HasPrefix(line, i.CmdPfx) {
		return line[len(i.CmdPfx):], true
	}

	if strings.HasPrefix(line, i.runtimeNick+": ") {
		return line[len(i.runtimeNick)+2:], true
	}

	return line, false
}

// HumanReadableSource takes an IRC userhost and returns just the nick
func (i *IRC) HumanReadableSource(source string) string {
	if out := ircutils.ParseUserhost(source).Nick; out != "" {
		return out
	}

	return source
}

// Status returns a human readable status string
func (i *IRC) Status() string {
	return fmt.Sprintf("IRC: Connected: %t Lag: %s", i.Connected.Get(), i.lag.Get())
}

// SendRaw sends a raw IRC line to the server
func (i *IRC) SendRaw(raw string) {
	if !i.Connected.Get() {
		i.log.Warn("cannot send a raw line when not connected")
		return
	}

	rawBytes := []byte(raw)

	n, err := i.write([]byte(raw), true)
	if n != len(rawBytes) {
		i.log.Warnf("Did not send enough bytes: %d != %d", n, len(rawBytes))
	}

	if err != nil {
		i.log.Warn("could not send message: ", err)
	}
}
