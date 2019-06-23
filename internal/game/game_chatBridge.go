package game

import (
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/ctcp"
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
	return dataForFmt{
		SourceNick:   uh.Nick,
		SourceUser:   uh.User,
		SourceHost:   uh.Host,
		Target:       target,
		MsgRaw:       msg,
		MsgEscaped:   ircfmt.Escape(msg),
		MsgMapped:    g.MapColours(msg),
		MsgStripped:  ircfmt.Strip(msg),
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

func (g *Game) OnJoin(source, channel string) {
	if !g.shouldBridge(channel) {
		return
	}
	g.checkError(g.SendFormattedLine(g.makeDataForFormat(source, channel, ""), g.chatBridge.format.join))
}

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

func (g *Game) OnNick(source, newnick string) {
	data := dataForNick{g.makeDataForFormat(source, "", ""), newnick}
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.nick))
}

func (g *Game) OnQuit(source, message string) {
	data := g.makeDataForFormat(source, "", message)
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.quit))
}

type dataForKick struct {
	dataForFmt
	Kickee string
}

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

func (g *Game) SendLineFromOtherGame(msg string, source interfaces.Game) {
	if !g.chatBridge.allowForwards {
		return
	}
	fmtData := dataForOtherGameFmt{msg, source.GetName()}
	g.checkError(g.SendFormattedLine(fmtData, g.chatBridge.format.external))
}

// SendFormattedLine executes the given format with the given data and sends the result to the process's STDIN
func (g *Game) SendFormattedLine(d interface{}, format util.Format) error {
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
