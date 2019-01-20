package main

import (
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/bot"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "github.com/chzyer/readline"
    "golang.org/x/sys/unix"
    "os"
    "os/signal"
    "strings"
)

func main() {
    rl, _ := readline.New("> ")
    log := botLog.NewLogger(botLog.FTimestamp, rl, "MAIN", 0)
    conf, err := config.GetConfig("config.xml")
    if err != nil {
        log.Panicf("could not read log file", err)
    }

    b := bot.NewBot(*conf, log.Clone().SetPrefix("BOT"))

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)

    go func() { sig := <-sigChan; b.Stop(fmt.Sprintf("Caught Signal: %s", sig)) }()

    go runCLI(b, rl)

    b.Run()
    fmt.Println()
}

func runCLI(b *bot.Bot, rl *readline.Instance) {

    lineChan := make(chan string)
    go func() {
        for {
            line, err := rl.Readline()
            if err != nil {
                close(lineChan)
                rl.Close()
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
