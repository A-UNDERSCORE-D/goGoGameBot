package main

import (
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/bot"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/config"
    "github.com/chzyer/readline"
    "log"
    "strings"
)

func main() {
    rl, _ := readline.New("> ")
    log.SetFlags(0)
    log.SetOutput(rl)
    defer rl.Close()

    conf, err := config.GetConfig("config.xml")
    if err != nil {
        panic(err)
    }

    b := bot.NewBot(*conf, log.New(rl, "[bot] ", 0))
    go func() {
        for {
            line, err := rl.Readline()
            if err != nil {
                // We're at an EOF, quit out
                return
            }
            splitLine := strings.Split(line, " ")

            b.CmdHandler.FireCommand(bot.CommandData{
                Command:   splitLine[0],
                Args:      splitLine[1:],
                IsFromIRC: false,
            })
        }
    }()

    b.Run()
    fmt.Println("done")
}
