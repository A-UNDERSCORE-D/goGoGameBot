package main

import (
    "fmt"
    "os"
    "os/signal"
    "strings"
    "time"

    "github.com/chzyer/readline"
    "github.com/spf13/pflag"
    "golang.org/x/sys/unix"

    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/bot"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

const (
    asciiArt = `
  ____  ____  ____ ____
 / ___|/ ___|/ ___| __ )
| |  _| |  _| |  _|  _ \
| |_| | |_| | |_| | |_) |
 \____|\____|\____|____/
`
    version = "0.1.1"
)

var (
    configFile = pflag.StringP("config", "c", "./config.xml", "Sets the config file location")
)

func main() {
    pflag.Parse()
    rl, _ := readline.New("> ")
    l := log.New(log.FTimestamp, rl, "MAIN", 0)

    for _, line := range strings.Split(asciiArt, "\n") {
        l.Info(line)
    }
    l.Infof("goGoGameBot version %s loading....", version)

    conf, err := config.GetConfig(*configFile)
    if err != nil {
        if err == config.ErrConfNotExist {
            l.Infof("Config file %s not found. Placing a default config there. Please set the configuration to your liking and restart gggb", *configFile)
            return
        }
        l.Panicf("could not read config file: %s", err)
    }
    b := bot.NewBot(*conf, l.Clone().SetPrefix("BOT"))

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)

    go func() { sig := <-sigChan; b.Stop(fmt.Sprintf("Caught Signal: %s", sig)) }()

    go runCLI(b, rl)
    if err := b.Run(); err != nil {
        l.Warnf("Got an error from bot on exit: %s", err)
    }
    b.StopAllGames()
    go func() {
        <-time.After(time.Second * 1)
        fmt.Println("Hang on close detected. forcing an exit")
        os.Exit(0)
    }()
    _ = rl.Close()
}

func runCLI(b *bot.Bot, rl *readline.Instance) {

    lineChan := make(chan string)
    go func() {
        for {
            line, err := rl.Readline()
            if err != nil {
                close(lineChan)
                b.Stop("SIGINT")
                return
            }
            lineChan <- line
        }
    }()

    for line := range lineChan {
        splitLine := strings.Split(line, " ")

        b.CmdHandler.FireCommand(&bot.CommandData{
            Command:   splitLine[0],
            Args:      splitLine[1:],
            IsFromIRC: false,
        })
    }
}
