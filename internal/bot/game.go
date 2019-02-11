package bot

import (
    "bufio"
    "bytes"
    "errors"
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
    "github.com/goshuirc/irc-go/ircfmt"
    "github.com/goshuirc/irc-go/ircmsg"
    "github.com/goshuirc/irc-go/ircutils"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "text/template"
    "time"
)

// TODO: Past x lines on stdout and stderr need to be stored, x being the largest requested by any GameRegexp

// Game is a representation of a game server
type Game struct {
    Name        string
    process     *process.Process
    regexpMutex sync.Mutex
    regexps     GameRegexpList
    log         *log.Logger
    adminChan   string
    logChan     string
    DumpStderr  bool
    DumpStdout  bool
    bot         *Bot

    /*chat stuff*/
    bridgeChat  bool
    bridgeChans []string
    bridgeFmt   util.Format
    colourMap   *strings.Replacer

    stdinChan chan []byte

    commandList []string

    killedByUs bool
}

// NewGame creates a game object for use in controlling a process
func NewGame(conf config.GameConfig, b *Bot) (*Game, error) {
    gameLog := b.Log.Clone().SetPrefix(conf.Name)
    if conf.WorkingDir == "" {
        fp, err := filepath.Abs(conf.Path)
        if err != nil {
            return nil, err
        }
        conf.WorkingDir = filepath.Dir(fp)
        gameLog.Infof("Unspecified working directory - inferred to %q", fp)
    }

    proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), conf.WorkingDir, gameLog.Clone())
    if err != nil {
        return nil, err
    }
    g := &Game{
        Name:      conf.Name,
        bot:       b,
        process:   proc,
        log:       gameLog,
        stdinChan: make(chan []byte, 50),
    }

    g.UpdateFromConf(conf)
    g.bot.HookPrivmsg(g.onPrivmsg) // TODO: This may end up with an issue if Game is ever deleted and the hook sits here. Probably need IDs or something
    //g.bot.CmdHandler.RegisterCommand(g.Name, g.commandHook, PriNorm,false)
    go g.watchStdinChan()
    g.bot.CmdHandler.RegisterCommand(fmt.Sprintf("%s_status", g.Name), func(data *CommandData) error {
        data.Reply(g.process.GetStatus())
        return nil
    }, PriNorm, false)
    return g, nil
}

func (g *Game) UpdateFromConf(conf config.GameConfig) {
    g.bridgeFmt = conf.BridgeFmt
    err := g.bridgeFmt.Compile(g.Name + "_bridge_format", nil)
    if err != nil {
        g.bot.Error(fmt.Errorf("could not compile template game %s: %s", g.Name, err))
    }

    g.adminChan = conf.AdminLogChan
    g.DumpStderr = conf.LogStderr
    g.DumpStdout = conf.LogStdout
    g.logChan = conf.Logchan
    g.bridgeChans = conf.BridgeChans
    g.bridgeChat = conf.BridgeChat // TODO: This causes a onetime race condition when reloading from IRC

    g.colourMap, err = util.MakeColourMap(conf.ColourMap.ToMap())
    if err != nil {
        g.bot.Error(err)
    }
    g.UpdateRegexps(conf.Regexps)
    g.process.UpdateCmd(conf.Path, strings.Split(conf.Args, " "), conf.WorkingDir)
    if !g.process.IsRunning() {
        if err := g.process.Reset(); err != nil {
            g.bot.Error(err)
        }
    }

    g.UpdateCommands(conf.Commands)

}

func (g *Game) UpdateCommands(conf []config.GameCommandConfig) {
    for _, c := range g.commandList {
        g.UnregisterCommand(c)
    }
    g.commandList = make([]string, 0)
    for _, c := range conf {
        g.RegisterCommand(c)
    }
}

type gameCommandData struct {
    IsFromIRC bool
    Args []string
    Source ircutils.UserHost
}

func (g *gameCommandData) ArgString() string {
    return strings.Join(g.Args, " ")
}

func (g *Game) UnregisterCommand(name string) {
    targetIdx := -1
    for i, n := range g.commandList {
        if n == name {
            targetIdx = i
            break
        }
    }

    if targetIdx == -1 {
        g.log.Warnf("attempt to remove unknown command %q", name)
        return
    }

    g.bot.CmdHandler.UnregisterCommand(name)
    g.commandList = append(g.commandList[:targetIdx], g.commandList[targetIdx+1:]...)
}

func (g *Game) RegisterCommand(conf config.GameCommandConfig) {
    if conf.Name == "" {
        g.bot.Error(errors.New("game: cannot create gamecommand with empty name"))
        return
    }
    templ, err := template.New(conf.Name).Funcs(util.TemplateUtilFuncs).Parse(conf.StdinFormat)
    if err != nil {
        g.bot.Error(fmt.Errorf("game: could not create GameCommand template: %s", err))
        return
    }
    resolvedName := strings.ToUpper(fmt.Sprintf("%s_%s", g.Name, conf.Name))
    g.log.Infof("registering command %q", resolvedName)
    g.bot.CmdHandler.RegisterCommand(
        resolvedName,
        func(data *CommandData) error {
            toSend := new(bytes.Buffer)
            uh, _ := data.UserHost()
            if err := templ.Execute(toSend, &gameCommandData{data.IsFromIRC, data.Args, uh}); err != nil {
                return err
            }

            g.stdinChan <- toSend.Bytes()
            return nil
        },
        PriNorm,
        conf.RequiresAdmin,
    )
    g.commandList = append(g.commandList, resolvedName)
}

// UpdateRegeps takes a config and updates all the available GameRegexps on its game object. This exists to facilitate
// runtime reloading of parts of the config
func (g *Game) UpdateRegexps(conf []config.GameRegexpConfig) {
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
    sort.Sort(g.regexps)
}

// Run starts the game and blocks until it completes
func (g *Game) Run() {
    g.sendToLogChan("starting")
    g.killedByUs = false
    if err := g.process.Start(); err != nil {
        g.bot.Error(err)
        return
    }
    g.startStdWatchers()

    if err := g.process.WaitForCompletion(); err != nil {
        if g.killedByUs {
            return
        }
        g.bot.Error(fmt.Errorf("[%s]: error on exit: %s", g.Name, err))
    }

    g.sendToLogChan("Process exited with " + g.process.GetReturnStatus())
    if err := g.process.Reset(); err != nil {
        g.bot.Error(err)
    }
}

func (g *Game) StopOrKillTimeout(timeout time.Duration) error {
    if !g.process.IsRunning() {
        return nil
    }
    g.sendToLogChan("stopping")
    g.killedByUs = true
    return g.process.StopOrKillTimeout(timeout)
}

// StopOrKill sends SIGINT to the running game, and after 30 seconds if the game has not closed on its own, it sends
// SIGKILL
func (g *Game) StopOrKill() error {
    return g.StopOrKillTimeout(time.Second * 30)
}

// StopOrKillWaitgroup is exactly the same as StopOrKill but it takes a waitgroup that is marked as done after the game
// has exited
func (g *Game) StopOrKillWaitgroup(wg *sync.WaitGroup) {
    if err := g.StopOrKillTimeout(time.Second * 30); err != nil {
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
        shouldEat, err := gRegexp.CheckAndExecute(line, stderr)
        if err != nil {
            g.bot.Error(err)
            continue
        }
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

type dataForFmt struct {
    SourceNick   string
    SourceUser   string
    SourceHost   string
    MsgRaw       string
    MsgEscaped   string
    MsgMapped    string
    Target       string
    MatchesStrip bool
}

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
    escapedLine := ircfmt.Escape(msg)
    err := g.SendBridgedLine(dataForFmt{
        SourceNick:   uh.Nick,
        SourceUser:   uh.User,
        SourceHost:   uh.Host,
        Target:       target,
        MsgRaw:       msg,
        MsgEscaped:   escapedLine,
        MsgMapped:    g.MapColours(msg),
        MatchesStrip: util.AnyMaskMatch(source, g.bot.Config.Strips),
    })

    if err != nil {
        bot.Error(err)
        return
    }
}

func (g *Game) SendBridgedLine(d dataForFmt) error {
    res, err := g.bridgeFmt.Execute(d)
    if err != nil {
        return err
    }
    if _, err := g.WriteString(res); err != nil {
        return err
    }
    return nil
}

func (g *Game) watchStdinChan() {
    for {
        toSend := <-g.stdinChan
        toSend = append(bytes.Trim(toSend, "\r\n"), '\n')
        if _, err := g.process.Write(toSend); err != nil {
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

/***util functions*****************************************************************************************************/

func (g *Game) MapColours(s string) string {
    if g.colourMap == nil {
        g.log.Warn("Colourmap is nil. returning stripped string instead")
        return ircfmt.Strip(s)
    }
    return g.colourMap.Replace(ircfmt.Escape(s))
}

func (g *Game) WriteString(s string) (n int, err error) {
    return g.Write([]byte(s))
}
