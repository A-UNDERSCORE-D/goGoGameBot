package main

import (
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/bot"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/cli"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/config"
    "github.com/chzyer/readline"
    "log"
)

var rl *readline.Instance

func init() {
    log.SetFlags( /*log.LstdFlags*/ 0)
    lrl, err := readline.New("> ")
    if err != nil {
        panic(err)
    }
    rl = lrl
    log.SetOutput(rl)
    cli.InitCLI(rl)

}

func main() {
    conf, err := config.GetConfig("config.xml")
    if err != nil {
        panic(err)
    }
    log.Print(conf)

    b := bot.NewBot(*conf, log.New(rl, "[bot]", 0))

    ch := bot.NewCommandHandler(b, "~")
    ch.RegisterCommand("test", func(data bot.CommandData) error {
        _, err := rl.Write([]byte("test"))
        return err
    }, bot.PriNorm)
    panic(b.Run())
}
