package bot

import (
    "bufio"
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/process"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "strings"
    "time"
)

// TODO: This needs a working directory etc on its process
type Game struct {
    Name       string
    process    *process.Process
    regexps    []*GameRegexp
    log        *botLog.Logger
    adminChan  string
    logChan    string
    DumpStderr bool
    DumpStdout bool
    bot        *Bot
}

func NewGame(conf config.Game, b *Bot) (*Game, error) {
    procL := b.Log.Clone() // Duplicate l for use elsewhere
    procL.SetPrefix(conf.Name)
    proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), procL)
    if err != nil {
        return nil, err
    }

    gL := b.Log.Clone()
    gL.SetPrefix(conf.Name)

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

    var gr []*GameRegexp

    for _, reConf := range conf.Regexps {
        r, err := NewGameRegexp(g, reConf)
        if err != nil {
            g.bot.Error(fmt.Errorf("could not create gameRegexp %s for game %s: %s", reConf.Name, g.Name, err))
            continue
        }
        gr = append(gr, r)
    }

    g.regexps = gr

    return g, nil
}

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

func (g *Game) StopOrKill() error {
    return g.process.StopOrKillTimeout(time.Second * 30)
}

func (g *Game) sendToLogChan(msg string) {
    g.bot.SendPrivmsg(g.logChan, fmt.Sprintf("[%s] %s", g.Name, msg))
}

func (g *Game) startStdWatchers() {
    go g.watchStd(false)
    go g.watchStd(true)
}

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

func (g *Game) handleOutput(line string, stderr bool) {
    pfx := "[STDOUT] "
    if stderr {
        pfx = "[STDERR] "
    }

    if (stderr && g.DumpStderr) || (!stderr && g.DumpStdout) {
        g.sendToLogChan(pfx + line)
    }

    g.log.Info(pfx, line)
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
