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

	g := &Game{
		name:      conf.Name,
		status:    normal,
		manager:   manager,
		Logger:    manager.Logger.Clone().SetPrefix(conf.Name),
		stdinChan: make(chan []byte),
	}
	go g.watchStdinChan()
	g.regexpManager = NewRegexpManager(g)
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
	join     util.Format
	part     util.Format
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
	statusMutex     sync.Mutex
	status          status
	autoRestart     int
	autoStart       bool
	regexpManager   *RegexpManager
	stdinChan       chan []byte
	controlChannels channelPair
	chatBridge      chatBridge
	allowForwards   bool
}

func (g *Game) setInternalStatus(status status) {
	g.statusMutex.Lock()
	g.status = status
	g.statusMutex.Unlock()
}

func (g *Game) getInternalStatus() status {
	g.statusMutex.Lock()
	defer g.statusMutex.Unlock()
	return g.status
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

		if shouldBreak || g.getInternalStatus() == killed || g.process.GetReturnCode() != 0 || g.autoRestart < 0 {
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
	g.setInternalStatus(normal)
	if err := g.process.Start(); err != nil {
		return false, err
	}
	g.monitorStdIO()

	if err := g.process.WaitForCompletion(); err != nil && g.getInternalStatus() != killed {
		return true, err
	}

	return true, nil
}

// UpdateFromConfig updates the game object with the data from the config object.
func (g *Game) UpdateFromConfig(conf config.Game) error {
	if conf.Name != g.GetName() {
		g.Warn("attempt to reload game with a config who's name does not match ours! bailing out of reload")
		return fmt.Errorf("invalid config name")
	}

	if conf.ControlChannels.Admin == "" || conf.ControlChannels.Msg == "" {
		g.Warn("cannot have an empty admin or msg channel. bailing out of reload")
		return fmt.Errorf("cannot have an empty admin or msg channel")
	}

	var wd string
	if conf.WorkingDir == "" {
		wd = path.Dir(conf.Path)
		g.Logger.Infof("game %q's working directory inferred to %q from binary path %q", g.GetName(), wd, conf.Path)
	}

	if err := g.regexpManager.UpdateFromConf(conf.Regexps); err != nil {
		return fmt.Errorf("could not update regepxs from config: %s", err)
	}

	cm, err := util.MakeColourMap(conf.ColourMap.ToMap())
	if err != nil {
		return fmt.Errorf("could not create colour map for game %s from config: %s", g, err)
	}

	if err := g.CompileFormats(&conf); err != nil {
		return fmt.Errorf("could not compile formats: %s", err)
	}

	_ = g.clearCommands() // This is going to error on first run or whenever we're first created, its fine

	for _, cmd := range conf.Commands {
		if err := g.registerCommand(cmd); err != nil {
			return err
		}
	}
	procArgs := strings.Split(conf.Args, " ")
	if g.process == nil {
		p, err := process.NewProcess(conf.Path, procArgs, wd, g.Logger.Clone())
		if err != nil {
			return err
		}
		g.process = p
	} else {
		g.process.UpdateCmd(conf.Path, procArgs, wd)
	}
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
	gf.join = f.Join
	gf.part = f.Part
	gf.nick = f.Nick
	gf.quit = f.Quit
	gf.kick = f.Kick
	gf.external = f.External
	g.Info("reload completed successfully")
	return nil
}

func (g *Game) CompileFormats(gameConf *config.Game) error {
	fmts := &gameConf.Chat.Formats
	if err := fmts.Message.Compile("message", false, nil); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "message", err)
	}
	root := fmts.Message.CompiledFormat
	if err := fmts.Join.Compile("join", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "join", err)
	}
	if err := fmts.Part.Compile("part", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "part", err)
	}
	if err := fmts.Nick.Compile("nick", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "nick", err)
	}
	if err := fmts.Quit.Compile("quit", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "quit", err)
	}
	if err := fmts.Kick.Compile("kick", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "kick", err)
	}
	if err := fmts.External.Compile("external", false, root); err != nil {
		return fmt.Errorf("could not compile format %s: %s", "external", err)
	}
	for _, v := range fmts.Extra {
		if err := v.Compile(v.Name, false, root); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) GetName() string {
	return g.name
}

func (g *Game) AutoStart() {
	if g.autoStart {
		go g.Run()
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
	g.setInternalStatus(killed)
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
