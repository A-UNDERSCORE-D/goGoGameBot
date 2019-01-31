package main

import (
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/bot"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
    "github.com/chzyer/readline"
    "golang.org/x/sys/unix"
    "os"
    "os/signal"
    "strings"
    "time"
)

func main() {
    rl, _ := readline.New("> ")
    l := log.New(log.FTimestamp, rl, "MAIN", 0)
    conf, err := config.GetConfig("config.xml")
    if err != nil {
        l.Panicf("could not read config file: %s", err)
    }

    b := bot.NewBot(*conf, l.Clone().SetPrefix("BOT"))

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)

    go func() { sig := <-sigChan; b.Stop(fmt.Sprintf("Caught Signal: %s", sig)) }()

    go runCLI(b, rl)

    b.Run()
    b.StopAllGames()
    go func(){
        <-time.After(time.Second * 1)
        fmt.Println("Hang on close detected. forcing an exit")
        os.Exit(0)
        }()
    rl.Close()
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
