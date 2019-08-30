package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/pflag"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/game"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/irc"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

const (
	asciiArt = `
  ____  ____  ____ ____
 / ___|/ ___|/ ___| __ )
| |  _| |  _| |  _|  _ \
| |_| | |_| | |_| | |_) |
 \____|\____|\____|____/
`
	version = "0.4.3"
)

var (
	configFile = pflag.StringP("config", "c", "./config.xml", "Sets the config file location")
	logger     *log.Logger
)

func main() {
	pflag.Parse()
	rl, _ := readline.New("> ")
	l := log.New(log.FTimestamp, rl, "MAIN", 0)
	logger = l

	for _, line := range strings.Split(asciiArt, "\n") {
		l.Info(line)
	}
	l.Infof("goGoGameBot version %s loading....", version)

	conf, err := config.GetConfig(*configFile)
	if err != nil {
		l.Panicf("could not read config file. Please ensure it exists and is correctly formatted (%s)", err)
	}

	conn, err := getConn(conf, l)
	if err != nil {
		l.Crit("could not create connection: ", err)
	}

	gm, err := game.NewManager(conf, conn, l.Clone().SetPrefix("GM"))
	if err != nil {
		l.Crit("could not create GameManager: ", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() { sig := <-sigChan; gm.Stop(fmt.Sprintf("Caught Signal: %s", sig), false) }()

	go runCLI(gm, rl)
	restart, err := gm.Run()
	if err != nil {
		l.Warnf("Got an error from bot on exit: %s", err)
	}

	if restart {
		execSelf()
	}

	go func() {
		// TODO: write to stdin?
		<-time.After(time.Second * 1)
		fmt.Println("Hang on close detected. forcing an exit")

		os.Exit(0)
	}()
	_ = rl.Close()
}

func execSelf() {
	executable, err := os.Executable()
	if err != nil {
		panic(err) // This should never fail and if it does we should explode violently
	}
	panic(syscall.Exec(executable, os.Args, []string{})) // This should never fail and if it does we should explode violently
}

type terminalUtil struct{}

//noinspection GoExportedElementShouldHaveComment
func (terminalUtil) AdminLevel(string) int { return 1337 }

//noinspection GoExportedElementShouldHaveComment
func (terminalUtil) SendMessage(_, message string) { logger.Info(message) }

//noinspection GoExportedElementShouldHaveComment
func (terminalUtil) SendNotice(_, message string) { logger.Info(message) }

func runCLI(gm *game.Manager, rl *readline.Instance) {
	lineChan := make(chan string, 1)
	go func() {
		for {
			line, err := rl.Readline()
			if err != nil {
				close(lineChan)
				gm.Stop("SIGINT", false)
				return
			}
			lineChan <- line
		}
	}()

	for line := range lineChan {
		gm.Cmd.ParseLine(line, true, "", "", terminalUtil{})
	}
}

func getConn(conf *config.Config, logger *log.Logger) (interfaces.Bot, error) {
	switch strings.ToLower(conf.ConnConfig.ConnType) {
	case "irc":
		return irc.New(conf.ConnConfig.Config, logger.Clone().SetPrefix("IRC"))
	default:
		return nil, fmt.Errorf("cannot resolve connType %q to a supported connection type", conf.ConnConfig.ConnType)
	}
}
