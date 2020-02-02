/*
Prog is the clientside component of networkTransport. It allows for network based games (this includes unix sockets)
*/
package main

import (
	"encoding/xml"
	"io/ioutil"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"sync"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/network"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/network/protocol"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"github.com/anmitsu/go-shlex"
	"github.com/spf13/pflag"
)

var (
	configPath = pflag.StringP(
		"config", "c", "./config.xml", "Sets the configuration file to use with this game instance",
	)
	waitForConf = pflag.BoolP("waitforconfig", "w", false, "if set, wait for a connection to give us a config")
	name        = pflag.StringP("name", "n", "game", "sets the name of this game, mostly for logging")
	logger      = log.New(log.FTimestamp, os.Stdout, "MAIN", log.TRACE)
	// TODO: Allow passing config by means of an RPC call? at least after the first one
)

/* func main2() {
	c, err := parseConfig("./config_test.xml")
	if err != nil {
		panic(err)
	}
	p, err := getProcess(c)
	if err != nil {
		panic(err)
	}

	proc := newProc(c, p, logger.Clone().SetPrefix("TEST"))

	one, two := net.Pipe()
	rpc.RegisterName("proc", proc)

	go rpc.ServeConn(one)
	client := rpc.NewClient(two)

	for {
		if err := client.Call("proc.RPCStartGame", struct{}{}, &struct{}{}); err != nil {
			panic(err)
		}

		lastLine := ""

		for {
			var out []string
			if err := client.Call("proc.GetStdout", lastLine, &out); err != nil {
				if err.Error() == "not running" {
					fmt.Println("expected")
					break
				}
				logger.Warnf("error while trying to get stdout: %s", err)
			}

			logger.Infof("lines returned: %#v", out)
			lastLine = out[len(out)-1]
		}

		fmt.Println("Sleeping, then restarting")
		time.Sleep(time.Second)
	}
}
*/

func main() {
	pflag.Parse()
	conf, err := parseConfig(*configPath)
	notNil("could not parse config: %s", err)
	p, err := getProcess(conf)
	notNil("could not create process: %s", err)
	listener, err := getListener(conf)
	notNil("could not get listener: %s", err)
	defer listener.Close()

	sigchan := make(chan os.Signal, 10)
	go func() { <-sigchan; listener.Close(); os.Exit(0) }()
	signal.Notify(sigchan, os.Interrupt)

	proc := &Proc{
		process:    p,
		conf:       conf,
		log:        logger.Clone().SetPrefix(*name),
		stdoutCond: sync.NewCond(&sync.Mutex{}),
		stderrCond: sync.NewCond(&sync.Mutex{}),
	}

	notNil("could not register RPC: %s", rpc.RegisterName(protocol.RPCName, proc))
	for {
		conn, err := listener.Accept()
		if err != nil {
			proc.log.Warnf("failed to accept connection: %s", err)
			return
		}
		go rpc.ServeConn(conn)
	}
}

func notNil(format string, err error) {
	if err != nil {
		logger.Critf(format, err)
	}
}

const maxCache = 1000 // Max size for caches before lines are dropped

func parseConfig(confPath string) (*network.Config, error) {
	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}
	conf := &network.Config{}

	err = xml.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func getProcess(conf *network.Config) (*process.Process, error) {
	workingDir := conf.WorkingDirectory
	if workingDir == "" {
		workingDir = path.Dir(*configPath)
		logger.Infof("working directory inferred to %s from %q", workingDir, *configPath)
	}

	procArgs, err := shlex.Split(conf.Args, true)
	if err != nil {
		return nil, err
	}

	return process.NewProcess(
		conf.Path,
		procArgs,
		workingDir,
		logger.Clone().SetPrefix(*name),
		conf.Environment,
		!conf.DontCopyEnv,
	)
}
