package bot

import (
    "bufio"
    "crypto/tls"
    "errors"
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
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

type RawChanPair struct {
    writeChan chan ircmsg.IrcMessage
    doneChan  chan bool
}

type Bot struct {
    Config    config.Config // Config for the IRC connection etc
    IrcConf   config.IRC
    sockMutex sync.Mutex
    sock      net.Conn
    Status    int                        // Current connection status
    DoneChan  chan bool                  // DoneChan will be closed when the connection is done. May be replaced by a waitgroup or other semaphore
    Log           *botLog.Logger         // Logger setup to have a prefix etc, for easy logging
    EventMgr      *eventmgr.EventManager // Main heavy lifter for the event system
    rawchansMutex sync.Mutex
    rawChans      map[string][]RawChanPair // rawChans holds channel pairs for use in blocking waits for lines
    capManager    *CapabilityManager
    CmdHandler    *CommandHandler
    Games         []*Game
}

func NewBot(conf config.Config) *Bot {
    b := &Bot{
        Config:   conf,
        IrcConf:  conf.Irc,
        Status:   DISCONNECTED,
        Log:      botLog.NewLogger("BOT", nil),
        EventMgr: new(eventmgr.EventManager),
        DoneChan: make(chan bool),
    }

    b.Init()
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
    b.capManager.requestCap(&Capability{Name: "sasl", Callback: b.saslHandler})
    b.capManager.NegotiateCaps()

    userMsg := util.MakeSimpleIRCLine("USER", b.IrcConf.Ident, "*", "*", b.IrcConf.Gecos)
    nickMsg := util.MakeSimpleIRCLine("NICK", b.IrcConf.Nick)

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
    b.Log.Infof("<< %s", string(line))
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
        b.Log.Warn("Did not write all bytes for a message")
    }
    return nil
}

func (b *Bot) readLoop() {
    scanner := bufio.NewScanner(b.sock)
    for scanner.Scan() {
        lineStr := scanner.Text()

        b.Log.Infof(">> %s", lineStr)

        line, err := ircmsg.ParseLine(lineStr)
        if err != nil {
            b.Log.Infof("[WARN] Discarding invalid line %q: %s", lineStr, err)
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
    upperCommand := strings.ToUpper(line.Command)
    go b.EventMgr.Dispatch("RAW_"+upperCommand, im)
    go b.EventMgr.Dispatch("RAW", im)
    go b.sendToRawChans(upperCommand, line)

}

func (b *Bot) HookRaw(cmd string, f func(ircmsg.IrcMessage, *Bot), priority int) {
    b.EventMgr.Attach(
        "RAW_"+cmd,
        func(s string, info eventmgr.InfoMap) {
            go f(info["line"].(ircmsg.IrcMessage), b)
        },
        priority,
    )
}

// Bot.Init sets up the default handlers and otherwise preps the bot to run
func (b *Bot) Init() {
    b.capManager = &CapabilityManager{bot: b}
    b.CmdHandler = NewCommandHandler(b, b.IrcConf.CommandPrefix)

    b.HookRaw("PING", onPing, PriHighest)
    b.HookRaw("001", onWelcome, PriNorm)

    b.EventMgr.Attach("ERR", func(s string, infoMaps eventmgr.InfoMap) {
        onError(infoMaps["Error"].(error), b)
    }, PriHighest)

    b.CmdHandler.RegisterCommand("RAW", rawCommand, PriNorm, true)
    b.CmdHandler.RegisterCommand("STARTGAME", b.StartGame, PriNorm, true)

    for _, gameConf := range b.Config.Games {
        g, err := NewGame(gameConf, b)
        if err != nil {
            b.Error(fmt.Errorf("could not create game %s: %s", gameConf.Name, err))
            continue
        }
        b.Games = append(b.Games, g)
    }
}

// Bot.Error dispatches an error event across the event manager with the given error
func (b *Bot) Error(err error) {
    b.EventMgr.Dispatch("ERR", eventmgr.InfoMap{"Error": err})
}

func (b *Bot) sendToRawChans(upperCommand string, line ircmsg.IrcMessage) {
    b.rawchansMutex.Lock()
    defer b.rawchansMutex.Unlock()
    chans, ok := b.rawChans[upperCommand]

    if !ok {
        return
    }

    for _, chanPair := range chans {
        // Just in case someone is sitting on this, that could be bad
        go func() {
            defer func() {
                err := recover()
                if err != nil {
                    b.Log.Warnf("[WARN] sendToRawChans lambda recovered panic: %s", err)
                }
            }()
            chanPair.writeChan <- line
        }()
    }
}

// Bot.GetRawChan returns a pair of channels, the first of which will receive ircmsg.IrcMessage as they come in
// and the second of which will
func (b *Bot) GetRawChan(command string) (<-chan ircmsg.IrcMessage, chan<- bool) {
    if b.rawChans == nil {
        b.rawChans = make(map[string][]RawChanPair)
    }

    command = strings.ToUpper(command)
    chanPair := RawChanPair{make(chan ircmsg.IrcMessage), make(chan bool)}
    b.rawchansMutex.Lock()
    defer b.rawchansMutex.Unlock()

    go func() {
        _, ok := <-chanPair.doneChan
        if ok {
            close(chanPair.doneChan)
        }
        close(chanPair.writeChan)
        b.rawchansMutex.Lock()
        defer b.rawchansMutex.Unlock()
        chanPairList := b.rawChans[command]

        for i, p := range chanPairList {
            if p == chanPair {
                chanPairList = append(chanPairList[:i], chanPairList[i+1:]...)
                break
            }
        }
    }()

    b.rawChans[command] = append(b.rawChans[command], chanPair)

    return chanPair.writeChan, chanPair.doneChan
}

// WaitForRaw waits for a single command and returns the line
func (b *Bot) WaitForRaw(command string) ircmsg.IrcMessage {
    w, d := b.GetRawChan(command)
    out := <-w
    close(d)
    return out
}

// GetMultiRawChan condenses multiple raw channels into one, allowing you to wait for any number of raw commands on
// a single channel
func (b *Bot) GetMultiRawChan(commands ... string) (<-chan ircmsg.IrcMessage, chan<- bool) {
    doneChan := make(chan bool)
    aggChan := make(chan ircmsg.IrcMessage)
    var donechans []chan<- bool
    for _, cmd := range commands {
        l, d := b.GetRawChan(cmd)
        donechans = append(donechans, d)
        go func() {
            for line := range l {
                aggChan <- line
            }
        }()
    }
    go func() {
        <-doneChan
        for _, c := range donechans {
            close(c)
        }
    }()

    return aggChan, doneChan
}

func (b *Bot) SendPrivmsg(target, msg string) {
    _ = b.WriteLine(util.MakeSimpleIRCLine("PRIVMSG", target, msg))
}

func (b *Bot) SendNotice(target, msg string) {
    me := target
    senpai := msg
    _ = b.WriteLine(util.MakeSimpleIRCLine("NOTICE", me, senpai))
}

func (b *Bot) Stop(quitMsg string) {
    if b.Status == DISCONNECTED {
        return
    }
    _ = b.WriteLine(util.MakeSimpleIRCLine("QUIT", quitMsg))
    b.WaitForRaw("ERROR")
    b.Status = DISCONNECTED
}


