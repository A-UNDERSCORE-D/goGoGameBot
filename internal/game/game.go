package game

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/anmitsu/go-shlex"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/mutexTypes"
)

const (
	normal = iota
	killed
	shutdown
)

// NewGame creates a new game from the given config
func NewGame(conf config.Game, manager *Manager) (*Game, error) {
	if conf.Name == "" {
		return nil, errors.New("cannot have an empty game name")
	}

	g := &Game{
		name:      conf.Name,
		status:    mutexTypes.Int{},
		manager:   manager,
		Logger:    manager.Logger.Clone().SetPrefix(conf.Name),
		stdinChan: make(chan []byte),
	}
	g.status.Set(normal)

	go g.watchStdinChan()

	g.regexpManager = NewRegexpManager(g)
	if err := g.UpdateFromConfig(conf); err != nil {
		return nil, err
	}

	for _, c := range g.chatBridge.channels {
		manager.bot.JoinChannel(c)
	}

	return g, nil
}

type formatSet struct {
	root     *template.Template
	message  *format.Format
	join     *format.Format
	part     *format.Format
	nick     *format.Format
	quit     *format.Format
	kick     *format.Format
	external *format.Format
	storage  *format.Storage
}

type channelPair struct {
	admin string
	msg   string
}

// Game represents a game server and its process
type Game struct {
	*log.Logger
	name            string
	process         *process.Process
	manager         *Manager
	status          mutexTypes.Int
	autoRestart     int
	autoStart       mutexTypes.Bool
	regexpManager   *RegexpManager
	stdinChan       chan []byte
	preRollRe       *regexp.Regexp
	preRollReplace  string
	controlChannels channelPair
	chatBridge      chatBridge
	allowForwards   bool
}

// Sentinel errors
var (
	ErrAlreadyRunning = errors.New("game is already running")
	ErrGameNotRunning = errors.New("game is not running")
)

// Run starts the given game if it is not already running. Note that this method blocks until the game exits, meaning
// you will probably want to use it in a goroutine
func (g *Game) Run() {
	for {
		if err := g.process.Reset(); err != nil {
			g.manager.Error(fmt.Errorf("error occurred while resetting process. not restarting: %s", err))
			break
		}

		shouldBreak := false
		cleanExit, err := g.runGame()

		if err == ErrAlreadyRunning {
			shouldBreak = true
		} else if err != nil && !strings.HasPrefix(err.Error(), "exit status") {
			g.manager.Error(err)
		}

		if !cleanExit {
			shouldBreak = true
		}

		g.sendToMsgChan(g.process.GetReturnStatus())

		if shouldBreak || g.status.Get() == killed || g.process.GetReturnCode() != 0 || g.autoRestart <= 0 {
			break
		}

		g.sendToMsgChan(fmt.Sprintf("Clean exit. Restarting in %d seconds", g.autoRestart))
		time.Sleep(time.Second * (time.Duration)(g.autoRestart))
	}
}

// runGame does the actual process handling for the game. It returns whether or not the process exited cleanly,
// and an error
func (g *Game) runGame() (bool, error) {
	if g.IsRunning() {
		g.sendToMsgChan("cannot start an already running game")
		return false, ErrAlreadyRunning
	}

	g.sendToMsgChan("starting")
	g.status.Set(normal)

	if err := g.process.Start(); err != nil {
		return false, err
	}

	g.monitorStdIO()

	if err := g.process.WaitForCompletion(); err != nil && g.status.Get() != killed {
		return true, err
	}

	return true, nil
}

func (g *Game) validateConfig(conf *config.Game) error {
	if conf.Name != g.GetName() {
		g.Warn("attempt to reload game with a config who's name does not match ours! bailing out of reload")
		return fmt.Errorf("invalid config name")
	}

	if conf.ControlChannels.Admin == "" || conf.ControlChannels.Msg == "" {
		g.Warn("cannot have an empty admin or msg channel. bailing out of reload")
		return fmt.Errorf("cannot have an empty admin or msg channel")
	}

	return nil
}

// UpdateFromConfig updates the game object with the data from the config object.
func (g *Game) UpdateFromConfig(conf config.Game) error {
	// Do as many of our checks as we can first before actually changing data, meaning that we can (hopefully)
	// prevent weird state in the case of an error
	if err := g.validateConfig(&conf); err != nil {
		return err
	}

	root := template.New(fmt.Sprintf("%s root", conf.Name))
	if err := g.compileFormats(&conf, root); err != nil {
		return fmt.Errorf("could not compile formats: %s", err)
	}

	if err := g.regexpManager.UpdateFromConf(conf.Regexps, root); err != nil {
		return fmt.Errorf("could not update regepxs from config: %s", err)
	}

	var preRollRe *regexp.Regexp

	if conf.PreRoll.Regexp != "" {
		re, err := regexp.Compile(conf.PreRoll.Regexp)
		if err != nil {
			return fmt.Errorf("could not compile preroll regexp: %w", err)
		}

		preRollRe = re
	}

	if err := g.setupTransformer(conf); err != nil {
		return fmt.Errorf("could not update game %q's config: %w", conf.Name, err)
	}

	wd := conf.WorkingDir
	if wd == "" {
		wd = path.Dir(conf.Path)
		g.Logger.Infof("game %q's working directory inferred to %q from binary path %q", g.GetName(), wd, conf.Path)
	}

	_ = g.clearCommands() // This is going to error on first run or whenever we're first created, its fine

	for _, cmd := range conf.Commands {
		if err := g.registerCommand(cmd); err != nil {
			return err
		}
	}

	procArgs, err := shlex.Split(conf.Args, true)
	if err != nil {
		return fmt.Errorf("could not parse game arguments: %w", err)
	}

	if g.process == nil {
		p, err := process.NewProcess(conf.Path, procArgs, wd, g.Logger.Clone(), conf.Env, !conf.DontCopyEnv)
		if err != nil {
			return err
		}

		g.process = p
	} else {
		g.process.UpdateCmd(conf.Path, procArgs, wd, conf.Env, !conf.DontCopyEnv)
	}

	g.autoStart.Set(conf.AutoStart)
	g.autoRestart = conf.AutoRestart // TODO: maybe check for 0 here

	// TODO: what are these used for?
	g.controlChannels.admin = conf.ControlChannels.Admin
	g.controlChannels.msg = conf.ControlChannels.Msg

	// Start of chat bridge configs
	g.chatBridge.shouldBridge = !conf.Chat.DontBridge
	g.chatBridge.dumpStdout = conf.Chat.DumpStdout
	g.chatBridge.dumpStderr = conf.Chat.DumpStderr
	g.chatBridge.allowForwards = !conf.Chat.DontAllowForwards
	g.chatBridge.channels = conf.Chat.BridgedChannels
	gf := &g.chatBridge.format
	f := &conf.Chat.Formats
	gf.message = f.Message
	gf.join = f.Join
	gf.part = f.Part
	gf.nick = f.Nick
	gf.quit = f.Quit
	gf.kick = f.Kick
	gf.external = f.External

	if gf.storage == nil {
		gf.storage = new(format.Storage)
	}

	g.preRollRe = preRollRe
	g.preRollReplace = conf.PreRoll.Replace

	g.chatBridge.format.root = root

	g.Info("reload completed successfully")

	return nil
}

func compileOrNil(targetFmt *format.Format, name string, root *template.Template) error {
	if targetFmt == nil || targetFmt.FormatString == "" {
		targetFmt = nil
		return nil
	}

	return targetFmt.Compile(name, root)
}

func (g *Game) compileFormats(gameConf *config.Game, root *template.Template) error {
	fmts := &gameConf.Chat.Formats

	const cantCompile = "could not compile format %s: %w"

	if err := compileOrNil(fmts.Message, "message", root); err != nil {
		return fmt.Errorf(cantCompile, "message", err)
	}

	if err := compileOrNil(fmts.Join, "join", root); err != nil {
		return fmt.Errorf(cantCompile, "join", err)
	}

	if err := compileOrNil(fmts.Part, "part", root); err != nil {
		return fmt.Errorf(cantCompile, "part", err)
	}

	if err := compileOrNil(fmts.Nick, "nick", root); err != nil {
		return fmt.Errorf(cantCompile, "nick", err)
	}

	if err := compileOrNil(fmts.Quit, "quit", root); err != nil {
		return fmt.Errorf(cantCompile, "quit", err)
	}

	if err := compileOrNil(fmts.Kick, "kick", root); err != nil {
		return fmt.Errorf(cantCompile, "kick", err)
	}

	if err := compileOrNil(fmts.External, "external", root); err != nil {
		return fmt.Errorf(cantCompile, "external", err)
	}

	for _, v := range fmts.Extra {
		if err := v.Compile(v.Name, root); err != nil {
			return err
		}
	}

	return nil
}

// GetName is a getter required by the interfaces.Game interface
func (g *Game) GetName() string {
	return g.name
}

// AutoStart checks if the game is marked as auto-starting, and if so, starts the game by starting Game.Run in a
// goroutine
func (g *Game) AutoStart() {
	if g.autoStart.Get() {
		go g.Run()
	}
}

func (g *Game) String() string {
	return fmt.Sprintf("game.Game at %p with manager %s", g, g.manager)
}

// StopOrKillTimeout sends SIGTERM to the running process. If the game is still running after the timeout has passed,
// the process is sent SIGKILL
func (g *Game) StopOrKillTimeout(timeout time.Duration) error {
	if !g.process.IsRunning() {
		if g.manager.status.Get() != shutdown {
			g.sendToMsgChan("cannot stop a non-running game")
		}

		return nil
	}

	g.sendToMsgChan("stopping")
	g.status.Set(killed)

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
