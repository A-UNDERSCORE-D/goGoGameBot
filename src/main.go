package main

import (
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/cli"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/command"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/irc/bot"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/process"
    "github.com/chzyer/readline"
    "log"
    "time"
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
    man := process.NewManager(
        rl,
        process.NewProcessMustSucceed("listener", "/usr/bin/ncat", []string{"127.0.0.1", "1337", "--listen"}, rl),
        process.NewProcessMustSucceed("client", "/usr/bin/ncat", []string{"127.0.0.1", "1337"}, rl),
    )

    h := command.Instance
    err := h.RegisterCommand("panic", func(string, []string, string, bool) bool {panic(""); return true })

    if err != nil {
        panic(err)
    }

    man.StartAllProcessesDelay(time.Millisecond * 10)
    man.WriteToProcess("client", "test!")

    b := bot.NewBot(bot.IRCConfig{Nick: "adtestbot", Ident: "adtest", Ssl: true, ServerHost: "irc.snoonet.org", ServerPort: "6697"}, rl)
    b.EventMgr.Attach("RAW_PRIVMSG", command.Instance.EventListener, bot.PriNorm)
    _ = command.Instance.RegisterCommand("test", func(cmd string, args []string, source string, fromIRC bool) bool {
        log.Print(cmd)
        log.Print(args)
        log.Print(source)
        log.Print(fromIRC)
        return true
        },
    )

    panic(b.Run())
    time.Sleep(time.Hour)
}
