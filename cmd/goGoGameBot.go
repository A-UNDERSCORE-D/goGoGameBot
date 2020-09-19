package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/spf13/pflag"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config/tomlconf"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/game"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/irc"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/nullconn"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/version"
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
)

var (
	configFile = pflag.StringP("config", "c", "./config.xml", "Sets the config file location")
	logger     *log.Logger
	traceLog   = pflag.Bool("trace", false, "enable trace logging (extremely verbose)")
	logFile    = pflag.StringP(
		"log-file", "l", "./%s.gggb.log",
		"sets the log file to be used. Must contain a %s for the date",
	)
	noLog = pflag.Bool("dont-log", false, "disables logging to disk")
)

func main() {
	pflag.Parse()

	rl, _ := readline.New("> ")
	lvl := log.DEBUG
	file, err := getLogFile(*logFile)

	if err != nil {
		panic(fmt.Sprintf("could not open log file: %s", err))
	}

	defer file.Close()

	writer := io.MultiWriter(rl, file)

	if *traceLog {
		lvl = log.TRACE
	}

	l := log.New(log.FTimestamp, writer, "MAIN", lvl)
	logger = l

	for _, line := range strings.Split(asciiArt, "\n") {
		l.Info(line)
	}

	l.Infof("goGoGameBot version %s loading....", version.Version)

	conf, err := tomlconf.GetConfig(*configFile)
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

	l.Info("Goodbye")

	if restart {
		execSelf()
	}

	go func() {
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
	// This should never fail and if it does we should explode violently
	panic(syscall.Exec(executable, os.Args, []string{}))
}

type terminalUtil struct{}

func (terminalUtil) AdminLevel(string) int { return 1337 }

func (terminalUtil) SendMessage(_, message string) { logger.Info(message) }

func (terminalUtil) SendNotice(_, message string) { logger.Info(message) }

func runCLI(gm *game.Manager, rl *readline.Instance) {
	lineChan := make(chan string, 1)

	go func() {
		for {
			line, err := rl.Readline()
			if err != nil {
				close(lineChan)
				gm.Stop("Internal Error", false)
				fmt.Println(err)

				return
			}
			lineChan <- line
		}
	}()

	for line := range lineChan {
		gm.Cmd.ParseLine(line, true, "", "", terminalUtil{})
	}
}

func getConn(conf *tomlconf.Config, logger *log.Logger) (interfaces.Bot, error) {
	switch strings.ToLower(conf.Connection.Type) {
	case "irc":
		return irc.New(conf.Connection, logger.Clone().SetPrefix("IRC"))
	case "null":
		return nullconn.New(logger.Clone().SetPrefix("null")), nil
	default:
		return nil, fmt.Errorf("cannot resolve connType %q to a supported connection type", conf.Connection.Type)
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func getLogFile(name string) (io.WriteCloser, error) {
	if *noLog {
		return nopWriteCloser{ioutil.Discard}, nil
	}

	curTime := time.Now().Format("02-01-2006")

	file, err := os.OpenFile(fmt.Sprintf(name, curTime), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	t := fmt.Sprintf("****Begin logging at %s", time.Now().String())

	if _, err := file.WriteString(t); err != nil {
		return nil, err
	}

	return file, nil
}
