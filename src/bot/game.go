package bot

import (
    "bufio"
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/process"
    "log"
    "strings"
)

// TODO: This needs a working directory etc on its process
type Game struct {
    Name       string
    process    *process.Process
    regexps    []*GameRegexp
    log        *log.Logger
    adminChan  string
    logChan    string
    DumpStderr bool
    DumpStdout bool
    bot        *Bot
    logPrefix  string
}

func NewGame(conf config.Game, b *Bot) (*Game, error) {
    procL := *b.Log // Duplicate l for use elsewhere
    procL.SetPrefix(fmt.Sprintf("[%s] [proc] ", conf.Name))
    proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), &procL)
    if err != nil {
        return nil, err
    }
    gL := *b.Log
    gL.SetPrefix("[" + conf.Name + "] ")

    g := &Game{
        Name:       conf.Name,
        bot:        b,
        process:    proc,
        log:        &gL,
        adminChan:  conf.AdminLogChan,
        DumpStderr: conf.LogStderr,
        DumpStdout: conf.LogStdout,
        logChan:    conf.Logchan,
        logPrefix:  "[" + conf.Name + "] ",
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
    if (stderr && g.DumpStderr) || (!stderr && g.DumpStdout) {
        pfx := "[STDOUT] "
        if stderr {
            pfx = "[STDERR] "
        }
        g.sendToLogChan(pfx + line)
    }

    for _, gRegexp := range g.regexps {
        shouldEat, res, err := gRegexp.CheckAndExecute(line, stderr)
        if err != nil {
            g.bot.Error(err)
            continue
        }

        g.log.Println(res)

        if shouldEat {
            break
        }
    }

}
