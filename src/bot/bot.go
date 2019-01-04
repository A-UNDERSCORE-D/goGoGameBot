package bot

import (
    "bufio"
    "crypto/tls"
    "errors"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/config"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
    "log"
    "net"
    "strings"
    "sync"
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

const (
    PriHighest = 16
    PriHigh    = 32
    PriNorm    = 48
    PriLow     = 64
    PriLowest  = 80
)

var (
    ErrNotConnected = errors.New("not connected to IRC")
)

func makeSimpleIRCLine(command string, args ...string) ircmsg.IrcMessage {
    return ircmsg.MakeMessage(nil, "", command, args...)
}

type Bot struct {
    Config    config.Config         // Config for the IRC connection etc
    IrcConf   config.IRC
    sockMutex sync.Mutex
    sock      net.Conn
    Status    int                    // Current connection status
    DoneChan  chan bool              // DoneChan will be closed when the connection is done. May be replaced by a waitgroup or other semaphore
    Log       *log.Logger            // Logger setup to have a prefix etc, for easy logging
    EventMgr  *eventmgr.EventManager // Main heavy lifter for the event system
}

func NewBot(conf config.Config, logger *log.Logger) *Bot {
    b := &Bot{
        Config:   conf,
        IrcConf:  conf.Irc,
        Status:   DISCONNECTED,
        Log:      logger,
        EventMgr: new(eventmgr.EventManager),
    }

    b.setupDefaultHandlers()
    return b
}

func (b *Bot) Run() error {
    if err := b.connect(); err != nil {
        return err
    }
    <-b.DoneChan
    return nil
}

func (b *Bot) connect() error {
    var sock net.Conn
    var err error

    if !b.Config.Irc.SSL {
        sock, err = net.Dial("tcp", b.IrcConf.Host+":"+b.IrcConf.Port)
    } else {
        sock, err = tls.Dial("tcp", b.IrcConf.Host+":"+b.IrcConf.Port, nil)
    }

    if err != nil {
        return err
    }
    b.sock = sock
    b.Status = CONNECTING

    go b.readLoop()
    userMsg := makeSimpleIRCLine("USER", b.IrcConf.Ident, "*", "*", b.IrcConf.Gecos)
    nickMsg := makeSimpleIRCLine("NICK", b.IrcConf.Nick)

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
    b.sockMutex.Lock()
    defer b.sockMutex.Unlock()
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
    b.EventMgr.Dispatch("RAW_"+strings.ToUpper(line.Command), im)
}

func (b *Bot) HookRaw(cmd string, f func(ircmsg.IrcMessage), priority int) {
    b.EventMgr.Attach(
        "RAW_"+cmd,
        func(s string, info eventmgr.InfoMap) {
            go f(info["line"].(ircmsg.IrcMessage))
        },
        priority,
    )
}

func (b *Bot) onPing(lineIn ircmsg.IrcMessage) {
    if err := b.WriteLine(makeSimpleIRCLine("PONG", lineIn.Params...)); err != nil {
        b.EventMgr.Dispatch("ERR", eventmgr.InfoMap{"error": err})
    }
}

func (b *Bot) onWelcome(lineIn ircmsg.IrcMessage) {
    // This should set a few things like max targets etc at some point.
    //lineIn := data["line"].(ircmsg.IrcMessage)
    b.Status = CONNECTED
}

func (b *Bot) setupDefaultHandlers() {
    b.HookRaw("PING", b.onPing, PriHighest)
    b.HookRaw("001", b.onWelcome, PriNorm)
}
