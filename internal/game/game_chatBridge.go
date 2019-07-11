package game

import (
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/ctcp"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/format"
)

type dataForFmt struct {
	SourceNick   string
	SourceUser   string
	SourceHost   string
	MsgRaw       string
	MsgEscaped   string
	MsgMapped    string
	MsgStripped  string
	Target       string
	MatchesStrip bool
	ExtraData    map[string]string
}

func (g *Game) makeDataForFormat(source string, target, msg string) dataForFmt {
	uh := ircutils.ParseUserhost(source)
	deZwsp := strings.ReplaceAll(msg, "\u200b", "")
	return dataForFmt{
		SourceNick:   uh.Nick,
		SourceUser:   uh.User,
		SourceHost:   uh.Host,
		Target:       target,
		MsgRaw:       msg,
		MsgEscaped:   ircfmt.Escape(deZwsp),
		MsgMapped:    g.MapColours(deZwsp),
		MsgStripped:  ircfmt.Strip(deZwsp),
		MatchesStrip: util.AnyMaskMatch(source, g.chatBridge.stripMasks),
		ExtraData:    make(map[string]string),
	}
}

func (g *Game) shouldBridge(target string) bool {
	if !g.chatBridge.shouldBridge || !g.process.IsRunning() || !strings.HasPrefix(target, "#") {
		return false
	}

	for _, c := range g.chatBridge.channels {
		if c == "*" || c == target {
			return true
		}
	}
	return false
}

type dataForPrivmsg struct {
	dataForFmt
	IsAction bool
}

// OnPrivmsg is a callback that is fired when a PRIVMSG is received from IRC
func (g *Game) OnPrivmsg(source, target, msg string) {
	if !g.shouldBridge(target) {
		return
	}

	isAction := false
	if out, err := ctcp.Parse(msg); err == nil {
		if out.Command != "ACTION" {
			return
		}
		msg = out.Arg
		isAction = true
	}
	data := dataForPrivmsg{g.makeDataForFormat(source, target, msg), isAction}
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.message))
}

// OnJoin is a callback that is fired when a user joins any channel
func (g *Game) OnJoin(source, channel string) {
	if !g.shouldBridge(channel) {
		return
	}
	g.checkError(g.SendFormattedLine(g.makeDataForFormat(source, channel, ""), g.chatBridge.format.join))
}

// OnPart is a callback that is fired when a user leaves a channel
func (g *Game) OnPart(source, target, message string) {
	if !g.shouldBridge(target) {
		return
	}
	g.checkError(g.SendFormattedLine(g.makeDataForFormat(source, target, message), g.chatBridge.format.part))
}

type dataForNick struct {
	dataForFmt
	NewNick string
}

// OnNick is a callback that is fired when a user changes their nickname
func (g *Game) OnNick(source, newnick string) {
	data := dataForNick{g.makeDataForFormat(source, "", ""), newnick}
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.nick))
}

// OnQuit is a callback that is fired when a user quits from IRC
func (g *Game) OnQuit(source, message string) {
	data := g.makeDataForFormat(source, "", message)
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.quit))
}

type dataForKick struct {
	dataForFmt
	Kickee string
}

// OnKick is a callback that is fired when a user kicks another user from the channel
func (g *Game) OnKick(source, channel, kickee, message string) {
	if !g.shouldBridge(channel) {
		return
	}
	data := dataForKick{g.makeDataForFormat(source, channel, message), kickee}
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.kick))
}

type dataForOtherGameFmt struct {
	Msg        string
	SourceGame string
}

// SendLineFromOtherGame Is a frontend for sending messages to a game from other games. If the game in source is the
// same as the current game, the name is switched to "LOCAL"
func (g *Game) SendLineFromOtherGame(msg string, source interfaces.Game) {
	if !g.chatBridge.allowForwards {
		return
	}
	name := "LOCAL"
	if source != g {
		name = source.GetName()
	}
	// Lets make sure to at least strip weird control characters when crossing games
	cleanMsg := strings.ReplaceAll(msg, "\u200b", "")
	fmtData := dataForOtherGameFmt{ircfmt.Strip(cleanMsg), name}
	g.checkError(g.SendFormattedLine(fmtData, g.chatBridge.format.external))
}

// SendFormattedLine executes the given format with the given data and sends the result to the process's STDIN
func (g *Game) SendFormattedLine(d interface{}, format format.Format) error {
	if !g.IsRunning() {
		return nil
	}

	res, err := format.Execute(d)
	if err != nil {
		return err
	}
	if len(res) == 0 {
		return nil
	}
	if _, err := g.WriteString(res); err != nil {
		return err
	}
	return nil
}
