package bot

import (
    "bufio"
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/process"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "sort"
    "strings"
    "sync"
    "time"
)

// TODO: This needs a working directory etc on its process
// TODO: Past x lines on stdout and stderr need to be stored, x being the largest requested by any GameRegexp
type Game struct {
    Name    string
    process *process.Process
    regexpMutex sync.Mutex
    regexps     GameRegexpList
    log         *botLog.Logger
    adminChan   string
    logChan     string
    DumpStderr  bool
    DumpStdout  bool
    bot         *Bot
}

// NewGame creates a game object for use in controlling a process
func NewGame(conf config.Game, b *Bot) (*Game, error) {
    procL := b.Log.Clone().SetPrefix(conf.Name) // Duplicate l for use elsewhere
    proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), procL)
    if err != nil {
        return nil, err
    }

    gL := b.Log.Clone().SetPrefix(conf.Name)

    g := &Game{
        Name:       conf.Name,
        bot:        b,
        process:    proc,
        log:        gL,
        adminChan:  conf.AdminLogChan,
        DumpStderr: conf.LogStderr,
        DumpStdout: conf.LogStdout,
        logChan:    conf.Logchan,
    }

    g.UpdateRegexps(conf.Regexps)

    return g, nil
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
