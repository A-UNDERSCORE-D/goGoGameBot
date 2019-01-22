package bot

import (
    "bufio"
    "bytes"
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/process"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "github.com/goshuirc/irc-go/ircmsg"
    "github.com/goshuirc/irc-go/ircutils"
    "github.com/pkg/errors"
    "sort"
    "strings"
    "sync"
    "text/template"
    "time"
)

// TODO: This needs a working directory etc on its process
// TODO: Past x lines on stdout and stderr need to be stored, x being the largest requested by any GameRegexp

// Game is a prepresentation
type Game struct {
    Name        string
    process     *process.Process
    regexpMutex sync.Mutex
    regexps     GameRegexpList
    log         *botLog.Logger
    adminChan   string
    logChan     string
    DumpStderr  bool
    DumpStdout  bool
    bot         *Bot

    /*chat stuff*/
    bridgeChat  bool
    bridgeChans []string
    bridgeFmt   *template.Template

    stdinChan chan []byte
}

// NewGame creates a game object for use in controlling a process
func NewGame(conf config.Game, b *Bot) (*Game, error) {
    proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), b.Log.Clone().SetPrefix(conf.Name))
    if err != nil {
        return nil, err
    }
    g := &Game{
        Name:      conf.Name,
        bot:       b,
        process:   proc,
        log:       b.Log.Clone().SetPrefix(conf.Name),
        stdinChan: make(chan []byte, 50),
    }

    g.UpdateFromConf(conf)
    g.bot.HookPrivmsg(g.onPrivmsg) // TODO: This may end up with an issue if Game is ever deleted and the hook sits here. Probably need IDs or something
    go g.watchStdinChan()
    return g, nil
}

func (g *Game) UpdateFromConf(conf config.Game) {
    bridgeFmt, err := template.New(conf.Name + "_bridge_format").Parse(conf.BridgeFmt)
    if err != nil {
        g.bot.Error(fmt.Errorf("could not compile template game %s: %s", g.Name, err))
    } else {
        g.bridgeFmt = bridgeFmt
    }

    g.adminChan = conf.AdminLogChan
    g.DumpStderr = conf.LogStderr
    g.DumpStdout = conf.LogStdout
    g.logChan = conf.Logchan
    g.bridgeChans = conf.BridgeChans
    g.bridgeChat = conf.BridgeChat
    g.UpdateRegexps(conf.Regexps)
    g.process.UpdateCmd(conf.Path, strings.Split(conf.Args, " "))
    if !g.process.IsRunning() {
        if err := g.process.Reset(); err != nil {
            g.bot.Error(err)
        }
    }
}

// UpdateRegeps takes a config and updates all the available GameRegexps on its game object. This exists to facilitate
// runtime reloading of parts of the config
func (g *Game) UpdateRegexps(conf []config.GameRegexp) {
    var newRegexps GameRegexpList

    for _, reConf := range conf {
        newRegexp, err := NewGameRegexp(g, reConf)
        if err != nil {
            g.bot.Error(fmt.Errorf("could not create gameRegexp %s for game %s: %s", reConf.Name, g.Name, err))
            continue
        }
        newRegexps = append(newRegexps, newRegexp)
        g.log.Debugf("added gameRegexp %q to game %q", newRegexp.Name, g.Name)
    }
    g.regexpMutex.Lock()
    defer g.regexpMutex.Unlock()
    g.regexps = newRegexps
    g.log.Debugf("pre-sorted regexp list:%#v", g.regexps)
    sort.Sort(g.regexps)
    g.log.Debugf("pst-sorted regexp list:%#v", g.regexps)
}

// Run starts the game and blocks until it completes
func (g *Game) Run() {
    if err := g.process.Start(); err != nil {
        g.bot.Error(err)
        return
    }
    g.startStdWatchers()

    if err := g.process.WaitForCompletion(); err != nil {
        g.bot.Error(err)
    }
    g.sendToLogChan("Process exited with " + g.process.GetProcStatus())
    if err := g.process.Reset(); err != nil {
        g.bot.Error(err)
    }
}

// StopOrKill sends SIGINT to the running game, and after 30 seconds if the game has not closed on its own, it sends
// SIGKILL
func (g *Game) StopOrKill() error {
    return g.process.StopOrKillTimeout(time.Second * 30)
}

// StopOrKillWaitgroup is exactly the same as StopOrKill but it takes a waitgroup that is marked as done after the game
// has exited
func (g *Game) StopOrKillWaitgroup(wg *sync.WaitGroup) {
    if err := g.process.StopOrKillTimeout(time.Second * 30); err != nil {
        g.bot.Error(err)
    }
    wg.Done()
}

// sendToLogChan sends the given message to the configured log channel for the game
func (g *Game) sendToLogChan(msg string) {
    g.bot.SendPrivmsg(g.logChan, fmt.Sprintf("[%s] %s", g.Name, msg))
}

func (g *Game) sendToAdminChan(msg string) {
    g.bot.SendPrivmsg(g.adminChan, fmt.Sprintf("[%s] %s", g.Name, msg))
}

// startStdWatches starts the read loops for stdout and stderr
func (g *Game) startStdWatchers() {
    go g.watchStd(false)
    go g.watchStd(true)
}

// watchStd watches the indicated std file for data and calls handle on the line
func (g *Game) watchStd(stderr bool) {
    var s *bufio.Scanner
    if stderr {
        s = bufio.NewScanner(g.process.Stderr)
    } else {
        s = bufio.NewScanner(g.process.Stdout)
    }

    for s.Scan() {
        line := s.Text()
        g.handleOutput(line, stderr)
    }
}

// handleOutput handles logging of stdout/err lines and running GameRegexps against them
func (g *Game) handleOutput(line string, stderr bool) {
    pfx := "[STDOUT] "
    if stderr {
        pfx = "[STDERR] "
    }

    if (stderr && g.DumpStderr) || (!stderr && g.DumpStdout) {
        g.sendToLogChan(pfx + line)
    }

    g.log.Info(pfx, line)
    g.regexpMutex.Lock()
    defer g.regexpMutex.Unlock()

    for _, gRegexp := range g.regexps {
        shouldEat, res, err := gRegexp.CheckAndExecute(line, stderr)
        if err != nil {
            g.bot.Error(err)
            continue
        }
        g.log.Info(res)
        if shouldEat {
            break
        }
    }
}

// Start of template funcs

func (g *Game) templSendToAdminChan(v ...interface{}) string {
    msg := fmt.Sprint(v...)
    g.sendToAdminChan(msg)
    return msg
}

func (g *Game) templSendToLogChan(v ...interface{}) string {
    msg := fmt.Sprint(v...)
    g.sendToLogChan(msg)
    return msg
}

func (g *Game) templSendPrivmsg(c string, v ...interface{}) (string, error) {
    if c == "" {
        return "", errors.New("cannot send to a nonexistant target")
    }
    msg := fmt.Sprint(v...)
    g.bot.SendPrivmsg(c, msg)
    return msg, nil
}

/**********************************************************************************************************************/

func (g *Game) onPrivmsg(source, target, msg string, originalLine ircmsg.IrcMessage, bot *Bot) {
    if !g.bridgeChat || !g.process.IsRunning() || !strings.HasPrefix(target, "#") {
        return
    }

    for _, c := range g.bridgeChans {
        if c == "*" || c == target {
            goto shouldForward
        }
    }
    return

shouldForward:
    uh := ircutils.ParseUserhost(source)
    buf := new(bytes.Buffer)

    err := g.bridgeFmt.Execute(buf, map[string]string{
        "source_nick": uh.Nick,
        "source_user": uh.User,
        "source_host": uh.Host,
        "msg":         msg,
        "target":      target,
    })
    g.Write(buf.Bytes())
    if err != nil {
        bot.Error(err)
    }
}

func (g *Game) watchStdinChan() {
    for {
        toSend := <-g.stdinChan
        toSend = append(bytes.Trim(toSend, "\r\n"), '\n')
        if _, err := g.process.Stdin.Write(toSend); err != nil {
            g.bot.Error(fmt.Errorf("could not write to stdin chan for %q: %s", g.Name, err))
        }
    }
}

func (g *Game) Write(p []byte) (n int, err error) {
    if !g.process.IsRunning() {
        return 0, errors.New("cannot write to a nonrunning game")
    }
    g.stdinChan <- p
    return len(p), nil
}
