package bot

import (
    "bufio"
    "crypto/tls"
    "errors"
    "github.com/chzyer/readline"
    "github.com/goshuirc/irc-go/ircmsg"
    "log"
    "net"
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
    Config   IRCConfig
    sock     net.Conn
    Status   int
    DoneChan chan bool
    Log      *log.Logger
}

func NewBot(config IRCConfig, rl *readline.Instance) *Bot {
    return &Bot{Config: config, Status: DISCONNECTED, Log: log.New(rl, "[bot] ", log.Flags())}
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
    switch line.Command {
    case "PING":
        if err := b.WriteLine(makeSimpleIRCLine("PONG", line.Params...)); err != nil {
            b.Log.Printf("could not write line: %s", err)
        }
    case "001":
        //b.onWelcome()
        b.Status = CONNECTED
    }
}
