package bot

import (
    "bufio"
    "crypto/tls"
    "errors"
    "github.com/chzyer/readline"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "log"
    "net"
    "strings"
)

const (
    // Connected means the bot has completed a connection to an IRC server
    CONNECTED = iota

    // Disconnected means the bot is not currently connected, could come from either a DC while connected or before the
    // connection is first made
    DISCONNECTED
    ERRORED

    // Connecting means the bot is in progress of connecting and negotiating with the target IRC server
    CONNECTING
)

const(
    PriLowest = 16
    PriLow = 32
    PriNorm = 48
    PriHigh = 64
    PriHighest = 80
)

var (
    ErrNotConnected = errors.New("not connected to IRC")
)

func makeSimpleIRCLine(command string, args ...string) ircmsg.IrcMessage {
    return ircmsg.MakeMessage(nil, "", command, args...)
}

type IRCConfig struct {
    Nick  string
    Ident string
    Gecos string

    Ssl        bool
    ServerHost string
    ServerPort string
}

type Bot struct {
    // Config for the IRC connection etc
    Config   IRCConfig
    sock     net.Conn
    // Current connection status
    Status   int
    // DoneChan will be closed when the connection is done. May be replaced by a waitgroup or other semaphore
    DoneChan chan bool
    // Logger setup to have a prefix etc, for easy logging
    Log      *log.Logger
    // Main heavy lifter for the event system
    EventMgr *eventmgr.EventManager
}

func NewBot(config IRCConfig, rl *readline.Instance) *Bot {
    b := &Bot{Config: config, Status: DISCONNECTED, Log: log.New(rl, "[bot] ", log.Flags())}
    b.EventMgr = &eventmgr.EventManager{}
    b.EventMgr.Attach("RAW_PING", b.onPing, PriNorm)
    b.EventMgr.Attach("RAW_001", b.onWelcome, PriNorm)
    return b
}

func (b *Bot) Run() error {
    if err := b.connect(); err != nil {
        return err
    }
    <- b.DoneChan
    return nil
}

func (b *Bot) connect() error {
    var sock net.Conn
    var err error

    if !b.Config.Ssl {
        sock, err = net.Dial("tcp", b.Config.ServerHost+":"+b.Config.ServerPort)
    } else {
        sock, err = tls.Dial("tcp", b.Config.ServerHost + ":" + b.Config.ServerPort, nil)
    }

    if err != nil {
        return err
    }
    b.sock = sock
    b.Status = CONNECTING

    go b.readLoop()
    userMsg := makeSimpleIRCLine("USER", b.Config.Ident, "*", "*", b.Config.Gecos)
    nickMsg := makeSimpleIRCLine("NICK", b.Config.Nick)

    if err := b.WriteLine(userMsg); err != nil {
        b.Status = ERRORED
        return err
    }
    if err := b.WriteLine(nickMsg); err != nil {
        b.Status = ERRORED
        return err
    }

    return nil
}

func (b *Bot) writeRaw(line []byte) (int, error) {
    b.Log.Printf("<< %s", string(line))
    return b.sock.Write(line)
}

func (b *Bot) WriteLine(line ircmsg.IrcMessage) error {
    if b.Status == DISCONNECTED {
        return ErrNotConnected
    }
    lb, err := line.LineBytes()
    if err != nil {
        return err
    }

    count, err := b.writeRaw(lb)
    if len(lb) != count {
        b.Log.Print("[WARN] Did not write all bytes for a message")
    }
    return nil
}
func (b *Bot) readLoop() {
    scanner := bufio.NewScanner(b.sock)
    for scanner.Scan() {
        lineStr := scanner.Text()

        b.Log.Printf(">> %s", lineStr)

        line, err := ircmsg.ParseLine(lineStr)
        if err != nil {
            b.Log.Printf("[WARN] Discarding invalid line %q: %s", lineStr, err)
            continue
        }

        b.HandleLine(line)
    }
    close(b.DoneChan)
}

func (b *Bot) HandleLine(line ircmsg.IrcMessage) {
    im := eventmgr.NewInfoMap()
    im["line"] = line
    im["bot"] = b
    b.EventMgr.Dispatch("RAW_" + strings.ToUpper(line.Command), im)
}

func (b *Bot) onPing(event string, data eventmgr.InfoMap) {
    lineIn := data["line"].(ircmsg.IrcMessage)
    if err := b.WriteLine(makeSimpleIRCLine("PONG", lineIn.Params...)); err != nil {
        b.EventMgr.Dispatch("error", eventmgr.InfoMap{"error": err})
    }
}

func (b *Bot) onWelcome(event string, data eventmgr.InfoMap) {
    // This should set a few things like max targets etc at some point.
    //lineIn := data["line"].(ircmsg.IrcMessage)
    b.Status = CONNECTED
}
