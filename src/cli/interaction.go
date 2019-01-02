package cli

import (
    "github.com/chzyer/readline"
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
    //split := strings.Split(line, " ")
}
