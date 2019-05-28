// 0+build ignore

package bot

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/goshuirc/irc-go/ircutils"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/command"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// TODO: Past x lines on stdout and stderr need to be stored, x being the largest requested by any GameRegexp

// Game is a representation of a game server
type Game struct {
	Name        string
	process     *process.Process
	regexpMutex sync.Mutex
	regexps     GameRegexpList
	log         *log.Logger
	adminChan   string
	logChan     string
	DumpStderr  bool
	DumpStdout  bool
	AutoStart   bool
	autoRestart bool
	bot         *Bot

	/*chat stuff*/
	bridgeChat           bool
	bridgeChans          []string
	bridgeFmt            util.Format
	joinPartFmt          util.Format
	forwardFromOthersFmt util.Format
	allowForwards        bool

	colourMap *strings.Replacer

	stdinChan chan []byte

	commandList []string

	killedByUs bool
}

// NewGame creates a game object for use in controlling a process
func NewGame(conf config.GameConfig, b *Bot) (*Game, error) {
	gameLog := b.Log.Clone().SetPrefix(conf.Name)
	if conf.WorkingDir == "" {
		fp, err := filepath.Abs(conf.Path)
		if err != nil {
			return nil, err
		}
		conf.WorkingDir = filepath.Dir(fp)
		gameLog.Infof("Unspecified working directory - inferred to %q", fp)
	}

	proc, err := process.NewProcess(conf.Path, strings.Split(conf.Args, " "), conf.WorkingDir, gameLog.Clone())
	if err != nil {
		return nil, err
	}
	g := &Game{
		Name:      conf.Name,
		bot:       b,
		process:   proc,
		log:       gameLog,
		stdinChan: make(chan []byte, 50),
	}

	g.UpdateFromConf(conf)
	g.bot.HookPrivmsg(g.onPrivmsg) // TODO: This may end up with an issue if Game is ever deleted and the hook sits here. Probably need IDs or something
	g.bot.HookRaw("JOIN", g.onJoinPart, PriNorm)
	g.bot.HookRaw("PART", g.onJoinPart, PriNorm)
	//g.bot.CmdHandler.RegisterCommand(g.Name, g.commandHook, PriNorm,false)
	go g.watchStdinChan()
	_ = g.bot.CommandManager.AddSubCommand(g.Name, "STATUS", 0, func(data command.Data) {
		data.SendTargetMessage(g.process.GetStatus())
	}, "returns the status of the supplied game")
	return g, nil
}

// UpdateFromConf updates the game with the given config object
func (g *Game) UpdateFromConf(conf config.GameConfig) {
	var err error
	g.bridgeFmt = conf.BridgeFmt
	g.CompileOrError(&g.bridgeFmt, "bridge_format", nil)
	g.joinPartFmt = conf.JoinPartFmt
	g.CompileOrError(&g.joinPartFmt, "join_part_format", nil)

	if conf.OtherForwardFmt.FormatString != "" {
		g.forwardFromOthersFmt = conf.OtherForwardFmt
		g.CompileOrError(&g.forwardFromOthersFmt, "forward_others_format", map[string]interface{}{"mapColours": g.MapColours})
		g.allowForwards = true
	}

	g.AutoStart = conf.AutoStart
	g.autoRestart = conf.RestartOnCleanExit

	g.adminChan = conf.AdminLogChan
	g.DumpStderr = conf.LogStderr
	g.DumpStdout = conf.LogStdout
	g.logChan = conf.LogChan
	g.bridgeChans = conf.BridgeChans
	g.bridgeChat = conf.BridgeChat // TODO: This causes a onetime race condition when reloading from IRC
	// TODO: Add status change formats to config

	g.colourMap, err = util.MakeColourMap(conf.ColourMap.ToMap())
	if err != nil {
		g.bot.Error(err)
	}
	g.UpdateRegexps(conf.Regexps)
	g.process.UpdateCmd(conf.Path, strings.Split(conf.Args, " "), conf.WorkingDir)
	if !g.process.IsRunning() {
		if err := g.process.Reset(); err != nil {
			g.bot.Error(err)
		}
	}

	g.UpdateCommands(conf.Commands)

}

// UpdateCommands replaces the game's commands with the ones in the passed slice
func (g *Game) UpdateCommands(conf []config.GameCommandConfig) {
	for _, c := range g.commandList {
		g.UnregisterCommand(c)
	}
	g.commandList = make([]string, 0)
	for _, c := range conf {
		g.RegisterCommand(c)
	}
}

type gameCommandData struct {
	IsFromIRC bool
	Args      []string
	Source    ircutils.UserHost
}

// ArgString returns the arguments on the object as a space separated string
func (g *gameCommandData) ArgString() string {
	return strings.Join(g.Args, " ")
}

// UnregisterCommand removes the given command name from the bot's command handler. If the given command does not exist
// it is logged and ignored
func (g *Game) UnregisterCommand(name string) {
	targetIdx := -1
	for i, n := range g.commandList {
		if n == name {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		g.log.Warnf("attempt to remove unknown command %q", name)
		return
	}

	g.bot.CommandManager.RemoveSubCommand(g.Name, name)
	g.commandList = append(g.commandList[:targetIdx], g.commandList[targetIdx+1:]...)
}

// RegisterCommand adds a command to the game's command list, if it errors, it passes those errors to the game's bot
func (g *Game) RegisterCommand(conf config.GameCommandConfig) {
	if conf.Name == "" {
		g.bot.Error(errors.New("game: cannot create gamecommand with empty name"))
		return
	}
	err := conf.StdinFormat.Compile(conf.Name, false)
	if err != nil {
		g.bot.Error(fmt.Errorf("game: could not create GameCommand for %s: template: %s", g.Name, err))
		return
	}
	resolvedName := strings.ToUpper(fmt.Sprintf("%s_%s", g.Name, conf.Name))
	g.log.Infof("registering command %q", resolvedName)

	adminLevel := 0
	if conf.RequiresAdmin {
		adminLevel = 1
	}

	g.bot.CommandManager.AddSubCommand(
		g.Name,
		conf.Name,
		adminLevel,
		func(data command.Data) {
			toSend, err := conf.StdinFormat.ExecuteBytes(&gameCommandData{data.IsFromIRC, data.Args, data.Source})
			if err != nil {
				return
			}
			g.stdinChan <- toSend
		},
		"no help available",
	)
	g.commandList = append(g.commandList, resolvedName)
}

// UpdateRegexps takes a config and updates all the available GameRegexps on its game object. This exists to facilitate
// runtime reloading of parts of the config
func (g *Game) UpdateRegexps(conf []config.GameRegexpConfig) {
	var newRegexps GameRegexpList

	for _, reConf := range conf {
		newRegexp, err := NewGameRegexp(g, reConf)
		if err != nil {
			g.bot.Error(fmt.Errorf("could not create gameRegexp %s for game %s: %s", reConf.Name, g.Name, err))
			continue
		}
		newRegexps = append(newRegexps, newRegexp)
		g.log.Debugf("added gameRegexp %q to game %q", newRegexp.Name, g.Name)
	}
	g.regexpMutex.Lock()
	defer g.regexpMutex.Unlock()
	g.regexps = newRegexps
	sort.Sort(g.regexps)
}

// Start of template funcs

func (g *Game) templSendToAdminChan(v ...interface{}) string {
	msg := fmt.Sprint(v...)
	g.sendToAdminChan(msg)
	return msg
}

func (g *Game) templSendToLogChan(v ...interface{}) string {
	msg := fmt.Sprint(v...)
	g.sendToLogChan(msg)
	return msg
}

func (g *Game) templSendPrivmsg(c string, v ...interface{}) (string, error) {
	if c == "" {
		return "", errors.New("cannot send to a nonexistent target")
	}
	msg := fmt.Sprint(v...)
	g.bot.SendPrivmsg(c, msg)
	return msg, nil
}

/**********************************************************************************************************************/

/***util functions*****************************************************************************************************/
