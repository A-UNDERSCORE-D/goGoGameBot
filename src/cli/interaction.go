package cli

import (
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/command"
    "github.com/chzyer/readline"
    "strings"
)

var rl *readline.Instance

func InitCLI(Lrl *readline.Instance) {
    rl = Lrl
    go runCLI()
}

func runCLI() {
    for {
        line, err := rl.Readline()
        if err != nil {
            // We're at an EOF, quit out
            return
        }
        handleLine(line)
    }
}

func handleLine(line string) {
    split := strings.Split(line, " ")
    command.Instance.HandleCommand(split[0], split[1:], "commandline.")
}
