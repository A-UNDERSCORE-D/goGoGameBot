package game

import (
	"strings"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
	"awesome-dragon.science/go/goGoGameBot/pkg/format"
	"awesome-dragon.science/go/goGoGameBot/pkg/format/transformer"
	"awesome-dragon.science/go/goGoGameBot/pkg/format/transformer/tokeniser"
	"awesome-dragon.science/go/goGoGameBot/pkg/util"
)

type chatBridge struct {
	shouldBridge  bool
	dumpStdout    bool
	dumpStderr    bool
	allowForwards bool
	stripMasks    []string
	channel       string
	format        formatSet
	transformer   transformer.Transformer
}

func (c *chatBridge) update(gc *tomlconf.Game, fmtSet *formatSet) {
	conf := gc.Chat
	c.shouldBridge = conf.BridgeChat
	c.dumpStdout = conf.DumpStdout
	c.dumpStderr = conf.DumpStderr
	c.allowForwards = conf.AllowForwards
	c.channel = conf.BridgedChannel

	c.format = *fmtSet

	if c.format.storage == nil {
		c.format.storage = new(format.Storage)
	}
}

type dataForFmt struct {
	game         *Game
	SourceRaw    string
	MsgRaw       string
	Target       string
	MatchesStrip bool
	ExtraData    map[string]string
	Storage      *format.Storage
}

// MsgEscaped returns the message in an escaped format
func (d *dataForFmt) MsgEscaped() string { return d.MsgRaw }

// MsgMapped returns the message as if it were mapped using the games transformer
func (d *dataForFmt) MsgMapped() string { return d.game.chatBridge.transformer.Transform(d.MsgRaw) }

// MsgStripped returns the message with all intermediate form codes stripped
func (d *dataForFmt) MsgStripped() string { return tokeniser.Strip(d.MsgRaw) }

// Source returns the source in a human readable form
func (d *dataForFmt) Source() string { return d.game.manager.bot.HumanReadableSource(d.SourceRaw) }

// MapString applies the game's transformer to the given string
func (d *dataForFmt) MapString(in ...string) string {
	return d.game.chatBridge.transformer.Transform(strings.Join(in, " "))
}

func (d *dataForFmt) InvokeTemplate(name string, args interface{}) (string, error) {
	out := new(strings.Builder)
	if err := d.game.chatBridge.format.root.ExecuteTemplate(out, name, args); err != nil {
		return "", err
	}

	return out.String(), nil
}

// This should always be given intermediate format data
func (g *Game) makeDataForFormat(source, target, msg string) *dataForFmt {
	deZwsp := strings.ReplaceAll(msg, "\u200b", "")

	return &dataForFmt{
		game:         g,
		SourceRaw:    source,
		Target:       target,
		MsgRaw:       deZwsp,
		MatchesStrip: util.AnyMaskMatch(source, g.chatBridge.stripMasks),
		ExtraData:    make(map[string]string),
		Storage:      g.chatBridge.format.storage,
	}
}

func (g *Game) shouldBridge(target string) bool {
	if !g.chatBridge.shouldBridge || !g.transport.IsRunning() {
		return false
	}

	return g.chatBridge.channel == "*" || g.chatBridge.channel == target
}

type dataForPrivmsg struct {
	dataForFmt
	IsAction bool
}

// OnMessage is a callback that is fired when a PRIVMSG is received from IRC
func (g *Game) OnMessage(source, target, msg string, isAction bool) {
	if !g.shouldBridge(target) || g.chatBridge.format.message == nil {
		return
	}

	data := dataForPrivmsg{*g.makeDataForFormat(source, target, msg), isAction}
	g.checkError(g.SendFormattedLine(&data, g.chatBridge.format.message))
}

// OnJoin is a callback that is fired when a user joins any channel
func (g *Game) OnJoin(source, channel string) {
	if !g.shouldBridge(channel) || g.chatBridge.format.join == nil {
		return
	}

	g.checkError(g.SendFormattedLine(g.makeDataForFormat(source, channel, ""), g.chatBridge.format.join))
}

// OnPart is a callback that is fired when a user leaves a channel
func (g *Game) OnPart(source, target, message string) {
	if !g.shouldBridge(target) || g.chatBridge.format.part == nil {
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
	if g.chatBridge.format.nick == nil {
		return
	}

	data := dataForNick{*g.makeDataForFormat(source, "", ""), newnick}
	g.checkError(g.SendFormattedLine(&data, g.chatBridge.format.nick))
}

// OnQuit is a callback that is fired when a user quits from IRC
func (g *Game) OnQuit(source, message string) {
	if g.chatBridge.format.quit == nil {
		return
	}

	data := g.makeDataForFormat(source, "", message)
	g.checkError(g.SendFormattedLine(&data, g.chatBridge.format.quit))
}

type dataForKick struct {
	dataForFmt
	Kickee string
}

// OnKick is a callback that is fired when a user kicks another user from the channel
func (g *Game) OnKick(source, channel, kickee, message string) {
	if !g.shouldBridge(channel) || g.chatBridge.format.kick == nil {
		return
	}

	data := dataForKick{*g.makeDataForFormat(source, channel, message), kickee}
	g.checkError(g.SendFormattedLine(&data, g.chatBridge.format.kick))
}

// SendLineFromOtherGame Is a frontend for sending messages to a game from other games. If the game in source is the
// same as the current game, the name is switched to "LOCAL"
func (g *Game) SendLineFromOtherGame(msg string, source interfaces.Game) {
	if !g.chatBridge.allowForwards || g.chatBridge.format.external == nil {
		g.Logger.Tracef(
			"early return from SendLineFromOtherGame: AF: %t externalF==Nil: %t src: %q msg: %q",
			g.chatBridge.allowForwards,
			g.chatBridge.format.external == nil,
			source.GetName(),
			msg,
		)

		return
	}

	name := "LOCAL"

	if source != g {
		name = source.GetName()
	}

	data := g.makeDataForFormat(name, "", util.StripAll(msg))
	g.checkError(g.SendFormattedLine(data, g.chatBridge.format.external))
}

// SendFormattedLine executes the given format with the given data and sends the result to the transport's STDIN
func (g *Game) SendFormattedLine(d interface{}, fmt *format.Format) error {
	if !g.IsRunning() {
		return nil
	}

	if fmt == nil {
		g.Logger.Warnf("game.SendFormattedLine passed a nil formatter")
		return nil
	}

	res, err := fmt.Execute(d)
	if err != nil {
		return err
	}

	if res == "" {
		return nil
	}

	if _, err := g.WriteString(res); err != nil {
		return err
	}

	return nil
}
