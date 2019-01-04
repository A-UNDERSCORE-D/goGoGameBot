package bot

import (
    "fmt"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/util"
    "github.com/goshuirc/eventmgr"
    "github.com/goshuirc/irc-go/ircmsg"
)

func onPing(lineIn ircmsg.IrcMessage, b *Bot) {
    if err := b.WriteLine(util.MakeSimpleIRCLine("PONG", lineIn.Params...)); err != nil {
        b.EventMgr.Dispatch("ERR", eventmgr.InfoMap{"error": fmt.Errorf("could not send pong: %s", err)})
    }
}

func onWelcome(lineIn ircmsg.IrcMessage, b *Bot) {
    // This should set a few things like max targets etc at some point.
    //lineIn := data["line"].(ircmsg.IrcMessage)
    b.Status = CONNECTED
}

func onError(err error, b *Bot) {
    b.Log.Printf("[WARN] error occured: %s", err)
}
