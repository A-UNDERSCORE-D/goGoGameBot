package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/goshuirc/irc-go/ircutils"
	"github.com/spf13/pflag"
	"golang.org/x/sys/unix"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/bot"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
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
	version = "0.3.4"
)

var (
	configFile = pflag.StringP("config", "c", "./config.xml", "Sets the config file location")
)

func main() {
	pflag.Parse()
	rl, _ := readline.New("> ")
	l := log.New(log.FTimestamp, rl, "MAIN", 0)

	for _, line := range strings.Split(asciiArt, "\n") {
		l.Info(line)
	}
	l.Infof("goGoGameBot version %s loading....", version)

	conf, err := config.GetConfig(*configFile)
	if err != nil {
		l.Panicf("could not read config file. Please ensure it exists and is correctly formatted (%s)", err)
	}
	b, err := bot.NewBot(*conf, l.Clone().SetPrefix("BOT"))
	if err != nil {
		l.Critf("error while creating bot: %s", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)

	go func() { sig := <-sigChan; b.Stop(fmt.Sprintf("Caught Signal: %s", sig), false) }()

	go runCLI(b, rl)
	err = b.Run()
	if err != nil && err != bot.ErrRestart {
		l.Warnf("Got an error from bot on exit: %s", err)
	}

	b.GameManager.StopAllGames()
	if err == bot.ErrRestart {
		ExecSelf()
	}

	go func() {
		<-time.After(time.Second * 1)
		fmt.Println("Hang on close detected. forcing an exit")

		os.Exit(0)
	}()
	_ = rl.Close()
}

func ExecSelf() {
	executable, err := os.Executable()
	if err != nil {
		panic(err) // This should never fail and if it does we should explode violently
	}
	panic(syscall.Exec(executable, os.Args, []string{})) // This should never fail and if it does we should explode violently
}

func runCLI(b *bot.Bot, rl *readline.Instance) {

	lineChan := make(chan string)
	go func() {
		for {
			line, err := rl.Readline()
			if err != nil {
				close(lineChan)
				b.Stop("SIGINT", false)
				return
			}
			lineChan <- line
		}
	}()

	for line := range lineChan {
		b.CommandManager.ParseLine(line, false, ircutils.UserHost{}, "")
	}
}
