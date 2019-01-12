package main

import (
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/bot"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "github.com/chzyer/readline"
    "golang.org/x/sys/unix"
    "log"
    "os"
    "os/signal"
    "strings"
)

func main() {
    rl, _ := readline.New("> ")
    botLog.InitLogger(rl, log.Ltime/* | log.Lshortfile*/)

    conf, err := config.GetConfig("config.xml")
    if err != nil {
        panic(err)
    }

    b := bot.NewBot(*conf)

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
