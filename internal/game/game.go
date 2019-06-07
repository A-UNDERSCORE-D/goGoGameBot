package game

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

const (
	normal status = iota
	killed
	shutdown
)

func NewGame(conf config.Game, manager *Manager) (*Game, error) {
	if conf.Name == "" {
		return nil, errors.New("cannot have an empty game name")
	}
	/*	cm, err := util.MakeColourMap(conf.ColourMap.ToMap())
		if err != nil {
			return nil, fmt.Errorf("could not build colour map for %s on %s: %s", conf.Name, manager, err)
		}

		fmts := conf.Chat.Formats
		for k, v := range map[string]util.Format{"normal": fmts.Normal, "joinpart": fmts.JoinPart, "nick": fmts.Nick, "quit": fmts.Quit, "kick": fmts.Kick, "external": fmts.External} {
			err := v.Compile(conf.Name+"_"+k, false)
			if err != nil {
				return nil, fmt.Errorf("could not compile %q game chat format for %s", k, conf.Name)
			}
		}*/

	g := &Game{
		name:   conf.Name,
		status: normal,
		/*		autoRestart:     conf.AutoRestart,
				autoStart:       conf.AutoStart,
				stdinChan:       make(chan []byte),
				controlChannels: channelPair{conf.ControlChannels.Admin, conf.ControlChannels.Msg},
				chatBridge: chatBridge{
					shouldBridge:  !conf.Chat.DontBridge,
					dumpStderr:    conf.Chat.DumpStderr,
					dumpStdout:    conf.Chat.DumpStdout,
					allowForwards: !conf.Chat.DontAllowForwards,
					stripMasks:    append(conf.Chat.StripMasks, manager.stripMasks...),
					channels:      conf.Chat.BridgedChannels,
					colourMap:     cm,
					format: formatSet{
						normal:   fmts.Normal,
						joinPart: fmts.JoinPart,
						nick:     fmts.Nick,
						quit:     fmts.Quit,
						kick:     fmts.Kick,
						external: fmts.External,
					},
				},*/
		manager: manager,
		Logger:  manager.Logger.Clone().SetPrefix("[" + conf.Name + "]"),
	}
	if err := g.UpdateFromConfig(conf); err != nil {
		return nil, err
	}

	return g, nil
}

type status int

type chatBridge struct {
	shouldBridge  bool
	dumpStdout    bool
	dumpStderr    bool
	allowForwards bool
	stripMasks    []string
	channels      []string
	format        formatSet
	colourMap     *strings.Replacer
}

type formatSet struct {
	message  util.Format
	joinPart util.Format
	nick     util.Format
	quit     util.Format
	kick     util.Format
	external util.Format
}

type channelPair struct {
	admin string
	msg   string
}

type Game struct {
	*log.Logger
	name            string
	process         *process.Process
	manager         *Manager
	status          status
	autoRestart     int
	autoStart       bool
	regexpManager   *RegexpManager
	stdinChan       chan []byte
	controlChannels channelPair
	chatBridge      chatBridge
	allowForwards   bool
}

var errAlreadyRunning = errors.New("game is already running")

func (g *Game) Run() {
	for {
		shouldBreak := false
		cleanExit, err := g.runGame()
		if err == errAlreadyRunning {
			shouldBreak = true
		} else if err != nil {
			g.manager.Error(err)
		}
		if !cleanExit {
			shouldBreak = true
		}

		if shouldBreak || g.status == killed || g.process.GetReturnCode() != 0 || g.autoRestart < 0 {
			break
		}

		g.sendToMsgChan(fmt.Sprintf("Clean exit. Restarting in %d seconds", g.autoRestart))
		if err := g.process.Reset(); err != nil {
			g.manager.Error(fmt.Errorf("error occurred while resetting process. not restarting: %s", err))
			break
		}
		time.Sleep(time.Second * (time.Duration)(g.autoRestart))
	}
}

func (g *Game) runGame() (bool, error) {
	if g.IsRunning() {
		g.sendToMsgChan("cannot start an already running game")
		return false, errAlreadyRunning
	}

	g.sendToMsgChan("starting")
	g.status = normal
	if err := g.process.Start(); err != nil {
		return false, err
	}
	g.monitorStdIO()

	if err := g.process.WaitForCompletion(); err != nil && g.status != killed {
		return true, err
	}

	return true, nil
}

// UpdateFromConfig updates the game object with the data from the config object. It is atomic--No changes will be applied
// if any pre-processing errors, and that error will be returned
func (g *Game) UpdateFromConfig(conf config.Game) error {
	if conf.Name != g.GetName() {
		g.Crit("attempt to reload game with a config who's name does not match ours! bailing out of reload")
		return fmt.Errorf("invalid config name")
	}
	var wd string
	if conf.WorkingDir == "" {
		wd = path.Dir(conf.Path)
		g.Logger.Infof("game %q's working directory inferred to %q from binary path %q", g.GetName(), wd, conf.Path)
	}
	if err := g.regexpManager.UpdateFromConf(conf.Regexps); err != nil {
		return err
	}

	cm, err := util.MakeColourMap(conf.ColourMap.ToMap())
	if err != nil {
		return fmt.Errorf("could not create colour map for game %s from config: %s", g, err)
	}
	gName := g.GetName()
	for _, err := range []error{
		conf.Chat.Formats.Normal.Compile(gName+"_Normal", false),
		conf.Chat.Formats.JoinPart.Compile(gName+"_JoinPart", false),
		conf.Chat.Formats.Nick.Compile(gName+"_Nick", false),
		conf.Chat.Formats.Quit.Compile(gName+"_Quit", false),
		conf.Chat.Formats.Kick.Compile(gName+"_Kick", false),
		conf.Chat.Formats.External.Compile(gName+"_External", false),
	} {
		if err != nil {
			return fmt.Errorf("could not compile all formats for game %s (other formats may also have errored): %s", g, err)
		}
	}

	g.process.UpdateCmd(conf.Path, strings.Split(conf.Args, " "), wd)
	g.autoStart = conf.AutoStart
	g.autoRestart = conf.AutoRestart
	g.controlChannels.admin = conf.ControlChannels.Admin
	g.controlChannels.msg = conf.ControlChannels.Msg

	// Start of chat bridge configs
	g.chatBridge.shouldBridge = !conf.Chat.DontBridge
	g.chatBridge.dumpStdout = conf.Chat.DumpStdout
	g.chatBridge.dumpStderr = conf.Chat.DumpStderr
	g.chatBridge.allowForwards = !conf.Chat.DontAllowForwards
	g.chatBridge.channels = conf.Chat.BridgedChannels
	g.chatBridge.colourMap = cm
	gf := &g.chatBridge.format
	f := &conf.Chat.Formats
	gf.message = f.Message
	gf.joinPart = f.JoinPart
	gf.nick = f.Nick
	gf.quit = f.Quit
	gf.kick = f.Kick
	gf.external = f.External
	g.Info("reload completed successfully")
	return nil
}

func (g *Game) GetName() string {
	return g.name
}

func (g *Game) AutoStart() {
	if g.autoStart {
		g.Run()
	}
}

func (g *Game) String() string {
	return fmt.Sprintf("game.Game at %p with manager %s", g, g.manager)
}

// StopOrKillTimeout sends SIGTERM to the running process. If the game is still running after the timeout has passed,
// the process is sent SIGKILL
func (g *Game) StopOrKillTimeout(timeout time.Duration) error {
	if !g.process.IsRunning() && g.manager.status != shutdown {
		g.sendToMsgChan("cannot stop a non-running game")
		return nil
	}
	g.sendToMsgChan("stopping")
	g.status = killed
	return g.process.StopOrKillTimeout(timeout)
}

// StopOrKill sends SIGINT to the running game, and after 30 seconds if the game has not closed on its own, it sends
// SIGKILL
func (g *Game) StopOrKill() error {
	return g.StopOrKillTimeout(time.Second * 30)
}

// StopOrKillWaitgroup is exactly the same as StopOrKill but it takes a waitgroup that is marked as done after the game
// has exited
func (g *Game) StopOrKillWaitgroup(wg *sync.WaitGroup) {
	g.checkError(g.StopOrKillTimeout(time.Second * 30))
	wg.Done()
}
