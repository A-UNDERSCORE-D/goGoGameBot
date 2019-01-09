package bot

import (
    "fmt"
    "git.fericyanide.solutions/A_D/goGoGameBot/src/util"
    "github.com/goshuirc/irc-go/ircmsg"
    "sync"
)

func onPing(lineIn ircmsg.IrcMessage, b *Bot) {
    if err := b.WriteLine(util.MakeSimpleIRCLine("PONG", lineIn.Params...)); err != nil {
        b.Error(fmt.Errorf("could not send pong: %s", err))
    }
}

func onWelcome(lineIn ircmsg.IrcMessage, b *Bot) {
    // This should set a few things like max targets etc at some point.
    //lineIn := data["line"].(ircmsg.IrcMessage)
    b.Status = CONNECTED
    for _, c := range b.IrcConf.JoinChans {
        _ = b.WriteLine(util.MakeSimpleIRCLine("JOIN", c.Name, c.Key))
    }
}

func onError(err error, b *Bot) {
    b.Log.Printf("[WARN] Error occured: %s", err)
}

const (
    auth          = "AUTHENTICATE"
    errDuringSasl = "could not complete SASL auth: %q. Falling back to PRIVMSG based auth"
    errSaslFailed = "sasl authentication failed. Falling back to PRIVMSG based auth (caused by %q)"
)

func (b *Bot) saslHandler(capability *Capability, _ ircmsg.IrcMessage, group *sync.WaitGroup) {
    defer group.Done()
    aggChan, aggDone := b.GetMultiRawChan(
        auth,
        util.RPL_LOGGEDIN,
        util.RPL_LOGGEDOUT,
        util.RPL_NICKLOCKED,
        util.RPL_SASLSUCCESS,
        util.RPL_SASLFAIL,
        util.RPL_SASLTOOLONG,
        util.RPL_SASLABORTED,
        util.RPL_SASLALREADY,
        util.RPL_SASLMECHS,
    )

    defer close(aggDone)
    // Request PLAIN authentication
    if err := b.WriteLine(util.MakeSimpleIRCLine(auth, "PLAIN")); err != nil {
        b.Error(fmt.Errorf(errDuringSasl, err))
        return
    }

rangeLoop:
    for line := range aggChan {
        switch line.Command {
        case auth:
            if line.Params[0] == "+" {
                authStr := util.GenerateSASLString(b.IrcConf.Nick, b.IrcConf.NSAuth.Nick, b.IrcConf.NSAuth.Password)
                _ = b.WriteLine(util.MakeSimpleIRCLine(auth, authStr))
            } else {
                b.Error(fmt.Errorf(errDuringSasl, line.SourceLine))
            }

        case util.RPL_LOGGEDIN, util.RPL_SASLSUCCESS:
            break rangeLoop

        case util.RPL_NICKLOCKED, util.RPL_SASLFAIL, util.RPL_SASLTOOLONG, util.RPL_SASLABORTED, util.RPL_SASLALREADY, util.RPL_SASLMECHS:
            b.Error(fmt.Errorf(errSaslFailed, line.SourceLine))
            break rangeLoop

        default:
            break rangeLoop
        }
    }
}

func (b *Bot) StartGame(data *CommandData) error {
    if len(data.Args) < 1 {
        if data.IsFromIRC {
            b.SendNotice(data.Source, "startgame requires an argument")
        } else {
            b.Log.Printf("startgame requires an argument")
        }
    }

    for _, g := range b.Games {
        if g.Name == data.Args[0] {
            go g.Run()
            break
        }
    }
    return nil
}
