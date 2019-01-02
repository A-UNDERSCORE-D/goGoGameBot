package main

import (
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/bot"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/cli"
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
    // This breaks occasionally. Its due to the time it takes for the kernel to allocate the socket for ncat etc
    // Its not an issue with the code AFAIK, the sleep in the start tends to fix it
    //man := process.NewManager(
    //    rl,
    //    process.NewProcessMustSucceed("listener", "/usr/bin/ncat", []string{"127.0.0.1", "1337", "--listen"}, rl),
    //    process.NewProcessMustSucceed("client", "/usr/bin/ncat", []string{"127.0.0.1", "1337"}, rl),
    //)

    //man.StartAllProcessesDelay(time.Millisecond * 10)
    //man.WriteToProcess("client", "test!")

    b := bot.NewBot(bot.IRCConfig{Nick: "adtestbot", Ident: "adtest", Ssl: true, ServerHost: "bot.snoonet.org", ServerPort: "6697"}, rl)
    ch := bot.NewCommandHandler(b, "~")
    ch.RegisterCommand("test", func(data bot.CommandData) error {
        _, err := rl.Write([]byte("test"))
        return err
    }, bot.PriNorm)
    panic(b.Run())
}
