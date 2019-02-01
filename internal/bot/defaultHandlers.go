package bot

import (
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
    "git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
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
    _ = b.WriteLine(util.MakeSimpleIRCLine("JOIN", b.IrcConf.AdminChan.Name, b.IrcConf.AdminChan.Key))
    for _, c := range b.IrcConf.JoinChans {
        _ = b.WriteLine(util.MakeSimpleIRCLine("JOIN", c.Name, c.Key))
    }
}

func onError(err error, b *Bot) {
    msg := fmt.Sprintf("Error occured: %s", err)
    b.Log.Warnf(msg)
    if b.Status == CONNECTED {
        b.SendPrivmsg(b.Config.Irc.AdminChan.Name, "[ERROR] "+msg)
    }
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
        b.Error(fmt.Errorf(errDuringSasl, err)) // TODO: This should setup a callback to run privmsg auth
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

func msgOrLog(data *CommandData, msg string) {
    if data.IsFromIRC {
        data.Bot.SendPrivmsg(data.Target, msg)
    } else {
        data.Bot.Log.Warn(msg)
    }
}

func (b *Bot) StartGame(data *CommandData) error {
    if len(data.Args) < 1 {
        msgOrLog(data, "startgame requires an argument")
        return nil
    }
    gameName := data.Args[0]
    g, _ := b.GetGameByName(gameName)
    if g == nil {
        msgOrLog(data, fmt.Sprintf("%q is an invalid game name", gameName))
        return nil
    }
    go g.Run()
    return nil
}

func (b *Bot) StopGame(data *CommandData) error {
    if len(data.Args) < 1 {
        msgOrLog(data, "stopgame requires an argument")
        return nil
    }
    gameName := data.Args[0]
    g, _ := b.GetGameByName(gameName)
    if g == nil {
        msgOrLog(data, fmt.Sprintf("%q is an invalid game name", gameName))
        return nil
    }
    return g.StopOrKill()
}

func reloadGameCmd(data *CommandData) error {
    conf, err := config.GetConfig("config.xml") // TODO: when flags are added this needs to read them.
    if err != nil {
        return err
    }
    data.Bot.reloadGames(conf.Games)
    return nil
}