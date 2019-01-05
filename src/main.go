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

    conf, err := config.GetConfig("config.xml")
    if err != nil {
        panic(err)
    }

    b := bot.NewBot(*conf, log.New(rl, "[bot] ", 0))
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
