package game

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/internal/transport"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/util"
	"awesome-dragon.science/go/goGoGameBot/pkg/format"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
	"awesome-dragon.science/go/goGoGameBot/pkg/mutexTypes"
)

const (
	normal = iota
	killed
	shutdown
)

// NewGame creates a new game from the given config
func NewGame(conf *tomlconf.Game, manager *Manager) (*Game, error) {
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

	manager.bot.JoinChannel(conf.Chat.BridgedChannel)

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

// Game represents a game server and its transport
type Game struct {
	*log.Logger
	name           string
	comment        string
	transport      transport.Transport
	manager        *Manager
	status         mutexTypes.Int
	autoRestart    int
	autoStart      mutexTypes.Bool
	regexpManager  *RegexpManager
	stdinChan      chan []byte
	preRollRe      *regexp.Regexp
	preRollReplace string
	chatBridge     *chatBridge
}

// Sentinel errors
var (
	ErrAlreadyRunning = errors.New("game is already running")
	ErrGameNotRunning = errors.New("game is not running")
)

func (g *Game) runStep() bool {
	g.sendToBridgedChannel("starting")
	g.status.Set(normal)

	start := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(2) // 1 for stdout, 1 for stderr

	go g.monitorStdIO(start, wg)

	code, humanStatus, err := g.transport.Run(start)

	wg.Wait()

	if err != nil && !(errors.Is(err, util.ErrorAlreadyRunning) || strings.HasPrefix(err.Error(), "exit status")) {
		return false
	}

	g.sendToBridgedChannel(humanStatus)

	if g.status.Get() == killed || code != 0 || g.autoRestart <= 0 {
		return false
	}

	return true
}

// Run starts the given game if it is not already running. Note that this method blocks until the game exits, meaning
// you will probably want to use it in a goroutine
func (g *Game) Run() error {
	for g.runStep() {
		g.sendToBridgedChannel(fmt.Sprintf("Clean exit. Restarting in %d seconds", g.autoRestart))
		time.Sleep(time.Second * time.Duration(g.autoRestart))
	}

	return nil
}

func (g *Game) validateConfig(conf *tomlconf.Game) error {
	if conf.Name != g.GetName() {
		g.Warn("attempt to reload game with a config who's name does not match ours! bailing out of reload")
		return fmt.Errorf("invalid config name")
	}

	if conf.Chat.BridgedChannel == "" {
		g.Warn("cannot have an empty bridged channel. bailing out of reload")
		return fmt.Errorf("cannot have an empty bridged channel")
	}

	return nil
}

// UpdateFromConfig updates the game object with the data from the config object.
func (g *Game) UpdateFromConfig(conf *tomlconf.Game) error {
	// Do as many of our checks as we can first before actually changing data, meaning that we can (hopefully)
	// prevent weird state in the case of an error
	if err := g.validateConfig(conf); err != nil {
		return err
	}

	g.comment = conf.Comment

	root := template.New(fmt.Sprintf("%s root", conf.Name))

	outFmts, err := g.compileFormats(conf, root)
	if err != nil {
		return fmt.Errorf("could not compile formats: %s", err)
	}

	if err := g.regexpManager.UpdateFromConf(conf.Regexps, root); err != nil {
		return fmt.Errorf("could not update regexps from config: %s", err)
	}

	var preRollRe *regexp.Regexp

	if conf.PreRoll.Regexp != "" {
		re, err := regexp.Compile(conf.PreRoll.Regexp)
		if err != nil {
			return fmt.Errorf("could not compile preroll regexp: %w", err)
		}

		preRollRe = re
	}

	if g.chatBridge == nil {
		g.chatBridge = new(chatBridge)
	}

	g.chatBridge.update(conf, outFmts)

	if err := g.setupTransformer(conf); err != nil {
		return fmt.Errorf("could not update game %q's config: %w", conf.Name, err)
	}

	_ = g.clearCommands() // This is going to error on first run or whenever we're first created, its fine

	for name, cmd := range conf.Commands {
		if err := g.registerCommand(name, cmd); err != nil {
			return err
		}
	}

	if g.transport == nil {
		t, err := transport.GetTransport(conf.Transport.Type, conf.Transport, g.Logger.Clone())
		if err != nil {
			return err
		}

		g.transport = t
	}

	// TODO: add a g.transport.Type() that lets me check if this changed ever

	if err := g.transport.Update(conf.Transport); err != nil {
		return err
	}

	g.Info("transport reloaded successfully")
	g.autoStart.Set(conf.AutoStart)
	g.autoRestart = conf.AutoRestart // TODO: maybe check for 0 here; and/or check for restart loop

	g.preRollRe = preRollRe
	g.preRollReplace = conf.PreRoll.Replace

	g.Info("reload completed successfully")

	return nil
}

// compileFormats compiles all the formats for the game. If one is nil....
func (g *Game) compileFormats(gameConf *tomlconf.Game, root *template.Template) (*formatSet, error) {
	fmts := gameConf.Chat.Formats

	const cantCompile = "could not compile format %s: %w"

	var (
		outFmts = new(formatSet)
	)

	compile := func(name string, fmtString *string, target **format.Format) error {
		if fmtString == nil {
			*target = nil // Just to be sure.
			return nil
		}

		*target = &format.Format{FormatString: *fmtString}

		return (*target).Compile(name, root)
	}

	if err := compile("message", fmts.Message, &outFmts.message); err != nil {
		return nil, fmt.Errorf(cantCompile, "message", err)
	}

	if err := compile("join", fmts.Join, &outFmts.join); err != nil {
		return nil, fmt.Errorf(cantCompile, "join", err)
	}

	if err := compile("part", fmts.Part, &outFmts.part); err != nil {
		return nil, fmt.Errorf(cantCompile, "part", err)
	}

	if err := compile("nick", fmts.Nick, &outFmts.nick); err != nil {
		return nil, fmt.Errorf(cantCompile, "nick", err)
	}

	if err := compile("quit", fmts.Quit, &outFmts.quit); err != nil {
		return nil, fmt.Errorf(cantCompile, "quit", err)
	}

	if err := compile("kick", fmts.Kick, &outFmts.kick); err != nil {
		return nil, fmt.Errorf(cantCompile, "kick", err)
	}

	if err := compile("external", fmts.External, &outFmts.external); err != nil {
		return nil, fmt.Errorf(cantCompile, "external", err)
	}

	for name, fmtStr := range fmts.Extra {
		// we dont need to actually return this, because its attached to root already
		extra := &format.Format{FormatString: fmtStr}
		if err := extra.Compile(name, root); err != nil {
			return nil, fmt.Errorf("could not compile extra format %q: %w", name, err)
		}
	}

	return outFmts, nil
}

// GetName is a getter required by the interfaces.Game interface
func (g *Game) GetName() string { return g.name }

// GetComment is a getter required by the interfaces.Game interface
func (g *Game) GetComment() string { return g.comment }

// AutoStart checks if the game is marked as auto-starting, and if so, starts the game by starting Game.Run in a
// goroutine
func (g *Game) AutoStart() {
	if g.autoStart.Get() {
		go func() {
			if err := g.Run(); err != nil {
				g.Logger.Warnf("could not run game: %s", err)
			}
		}()
	}
}

func (g *Game) String() string {
	return fmt.Sprintf("game.Game at %p with manager %s", g, g.manager)
}

// StopOrKillTimeout sends SIGTERM to the running transport. If the game is still running after the timeout has passed,
// the transport is sent SIGKILL
func (g *Game) StopOrKillTimeout(timeout time.Duration) error {
	if !g.transport.IsRunning() {
		if g.manager.status.Get() != shutdown {
			g.sendToBridgedChannel("cannot stop a non-running game")
		}

		return nil
	}

	g.sendToBridgedChannel("stopping")
	g.status.Set(killed)

	return g.transport.StopOrKillTimeout(timeout)
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
