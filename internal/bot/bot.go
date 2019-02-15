package bot

import (
    "bufio"
    "crypto/tls"
    "errors"
    "fmt"
    "net"
    "runtime/debug"
    "strings"
    "sync"

    "github.com/goshuirc/irc-go/ircmsg"

    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/event"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
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
    Config        config.Config            // Config for the IRC connection etc
    IrcConf       config.BotConfig         // Easier access to the IRC section of the config
    sockMutex     sync.Mutex               // Mutex for the IRC socket
    sock          net.Conn                 // IRC socket
    Status        int                      // Current connection status
    DoneChan      chan bool                // DoneChan will be closed when the connection is done. May be replaced by a waitgroup or other semaphore
    Log           *log.Logger              // Logger setup to have a prefix etc, for easy logging
    EventMgr      *event.Manager           // Main heavy lifter for the event system
    rawchansMutex sync.Mutex               // Mutex protecting the rawChans map
    rawChans      map[string][]RawChanPair // rawChans holds channel pairs for use in blocking waits for lines
    capManager    *CapabilityManager       // Manager for IRCv3 capabilities
    CmdHandler    *CommandHandler          // Handler for irc (or commandline) commands
    Games         []*Game                  // Games loaded onto the bot
    GamesMutex    sync.Mutex               // Mutex protecting the games slice
}

func NewBot(conf config.Config, logger *log.Logger) *Bot {
    b := &Bot{
        Config:   conf,
        IrcConf:  conf.Irc,
        Status:   DISCONNECTED,
        Log:      logger,
        EventMgr: new(event.Manager),
        DoneChan: make(chan bool),
    }

    b.Init()
    return b
}

/***Start of funcs for upper level control***************************************************************************/

// Run starts the bot and lets it connect to IRC. It blocks until the IRC server connection is closed
func (b *Bot) Run() error {
    b.Log.Infof("Connecting to %s on port %s with ssl: %t", b.IrcConf.Host, b.IrcConf.Port, b.IrcConf.SSL)
    if err := b.connect(); err != nil {
        return err
    }
    <-b.DoneChan
    if b.Status != DISCONNECTED {
        b.Status = DISCONNECTED
    }
    return nil
}

// Stop makes the bot quit out and stop all its games
func (b *Bot) Stop(quitMsg string) {
    b.Log.Info("stop requested: ", quitMsg)
    b.StopAllGames()
    if b.Status == DISCONNECTED {
        return
    }

    _ = b.WriteLine(util.MakeSimpleIRCLine("QUIT", quitMsg))
    b.WaitForRaw("ERROR")
    b.Status = DISCONNECTED
}

func (b *Bot) stopCmd(data *CommandData) error {
    if data.ArgString() == "" {
        b.Stop("Quit requested")
    } else {
        b.Stop(data.ArgString())
    }
    return nil
}

// Init sets up the default handlers and otherwise preps the bot to run
func (b *Bot) Init() {
    b.capManager = &CapabilityManager{bot: b}
    b.CmdHandler = NewCommandHandler(b, b.IrcConf.CommandPrefix)

    b.HookRaw("PING", onPing, PriHighest)
    b.HookRaw("001", onWelcome, PriNorm)

    b.EventMgr.Attach("ERR", func(s string, maps event.ArgMap) {
        onError(maps, b)
    }, PriHighest)

    b.CmdHandler.RegisterCommand("RAW", rawCommand, PriNorm, true)
    b.CmdHandler.RegisterCommand("STARTGAME", b.StartGame, PriNorm, true)
    b.CmdHandler.RegisterCommand("STOPGAME", b.StopGame, PriNorm, true)
    b.CmdHandler.RegisterCommand("RELOADGAMES", reloadGameCmd, PriNorm, true)
    b.CmdHandler.RegisterCommand("STOP", b.stopCmd, PriHighest, true)
    b.CmdHandler.RegisterCommand("STATUS", func(data *CommandData) error {
        data.Reply("Main stats: " + util.GetHostStats())
        data.Bot.GamesMutex.Lock()
        defer data.Bot.GamesMutex.Unlock()
        for _, g := range data.Bot.Games {
            data.Reply(fmt.Sprintf("[%s] %s", g.Name, g.process.GetStatus()))
        }
        return nil
    }, PriNorm, true)
    b.reloadGames(b.Config.Games)
}

// connect opens a socket to the IRC server specified and handles basic registration and SASL auth
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

/***start of write- functions for accessing the socket*****************************************************************/

// WriteRaw writes bytes directly to the IRC server's socket, it also handles synchronisation and logging of outgoing
// lines
func (b *Bot) writeRaw(line []byte) (int, error) {
    b.sockMutex.Lock()
    defer b.sockMutex.Unlock()
    b.Log.Infof("<< %s", string(line))
    return b.sock.Write(line)
}

// WriteLine writes an ircmsg.IrcMessage to the connected IRC server
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

/***start of read oriented functions for accessing the socket**********************************************************/

// readLoop is the main listener loop for lines coming from the socket
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

// HandleLine is the main handler for raw lines coming from IRC
func (b *Bot) HandleLine(line ircmsg.IrcMessage) {
    im := event.ArgMap{}
    im["line"] = line
    im["bot"] = b
    upperCommand := strings.ToUpper(line.Command)
    go b.EventMgr.Dispatch("RAW_"+upperCommand, im)
    go b.EventMgr.Dispatch("RAW", im)
    go b.sendToRawChans(upperCommand, line)

}

/***start of util functions********************************************************************************************/

// Error dispatches an error event across the event manager with the given error
func (b *Bot) Error(err error) {
    b.EventMgr.Dispatch("ERR", event.ArgMap{"Error": err, "trace": debug.Stack()})
}

/***start of hook oriented functions***********************************************************************************/

type HookFunc func(ircmsg.IrcMessage, *Bot)

// HookRaw hooks a callback function onto a raw line. The callback given is launched in a goroutine.
func (b *Bot) HookRaw(cmd string, f HookFunc, priority int) {
    b.EventMgr.Attach(
        "RAW_"+cmd,
        func(s string, info event.ArgMap) {
            go f(info["line"].(ircmsg.IrcMessage), b)
        },
        priority,
    )
}

type PrivmsgFunc func(source, target, message string, originalLine ircmsg.IrcMessage, bot *Bot)

// HookPrivmsg hooks a callback to all PRIVMSG lines. The callback is launched in a goroutine.
func (b *Bot) HookPrivmsg(f PrivmsgFunc) {
    b.HookRaw("PRIVMSG",
        func(line ircmsg.IrcMessage, bot *Bot) {
            go f(line.Prefix, line.Params[0], line.Params[1], line, b)
        },
        DEFAULTPRIORITY,
    )
}

// sendToRawChans writes to all raw channels waiting on a given command
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
func (b *Bot) GetMultiRawChan(commands ...string) (<-chan ircmsg.IrcMessage, chan<- bool) {
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

/***start of send- style functions*************************************************************************************/

// SendPrivmsg sends a standard IRC message to the target. The target can be either a channel or a nickname
func (b *Bot) SendPrivmsg(target, msg string) {
    for _, v := range strings.Split(msg, "\n") {
        _ = b.WriteLine(util.MakeSimpleIRCLine("PRIVMSG", target, v))
    }
}

// SendNotice sends a notice to the target. The target can be either a channel or a nickname
func (b *Bot) SendNotice(target, msg string) {
    me := target
    for _, senpai := range strings.Split(msg, "\n") {
        _ = b.WriteLine(util.MakeSimpleIRCLine("NOTICE", me, senpai))
    }
}

/***start of game control functions***********************************************************************************/

// reloadGames reloads the data on all games using the given config slice
func (b *Bot) reloadGames(conf []config.GameConfig) { // TODO: Removing games that no longer exist?
    for _, gameConf := range conf {
        currentGame, idx := b.GetGameByName(gameConf.Name)
        if idx == -1 {
            g, err := NewGame(gameConf, b)
            if err != nil {
                b.Error(err)
                continue
            }
            b.addGame(g)

        } else {
            currentGame.UpdateFromConf(gameConf)
        }
    }
}

func (b *Bot) addGame(game *Game) {
    b.GamesMutex.Lock()
    defer b.GamesMutex.Unlock()
    b.Games = append(b.Games, game)

}

// GetGameByName returns the game with the given name, and the index for the game in the bot's game slice.
// If no game was found, the pointer will be nil, and the index will be -1
func (b *Bot) GetGameByName(name string) (*Game, int) {
    b.GamesMutex.Lock()
    defer b.GamesMutex.Unlock()
    for i, g := range b.Games {
        if g.Name == name {
            return g, i
        }
    }
    return nil, -1
}

func (b *Bot) StopAllGames() {
    wg := new(sync.WaitGroup)
    for _, g := range b.Games {
        wg.Add(1)
        go g.StopOrKillWaitgroup(wg)
    }
    wg.Wait()
}
