package cli

import (
    "git.fericyanide.solutions/A_D/goGoGameBot/src/bot"
    "github.com/chzyer/readline"
    "strings"
)

var rl *readline.Instance

func InitCLI(Lrl *readline.Instance, b *bot.Bot) {
    rl = Lrl
    go runCLI(b)
}

func runCLI(b *bot.Bot) {
    for {
        line, err := rl.Readline()
        if err != nil {
            // We're at an EOF, quit out
            return
        }
        handleLine(line, b)
    }
}

func handleLine(line string, b *bot.Bot) {

    splitLine := strings.Split(line, " ")

    b.CmdHandler.FireCommand(bot.CommandData{
        Command:splitLine[0],
        Args: splitLine[1:],
        IsFromIRC: false,
    })
}
