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

	"awesome-dragon.science/go/goGoGameBot/internal/process"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/network"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/network/protocol"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
	"github.com/anmitsu/go-shlex"
	"github.com/spf13/pflag"
)

var (
	configPath = pflag.StringP(
		"config", "c", "./config.xml", "Sets the configuration file to use with this game instance",
	)
	// waitForConf = pflag.BoolP("waitforconfig", "w", false, "if set, wait for a connection to give us a config")
	name   = pflag.StringP("name", "n", "game", "sets the name of this game, mostly for logging")
	logger = log.New(log.FTimestamp, os.Stdout, "MAIN", log.TRACE)
	// TODO: Allow passing config by means of an RPC call? at least after the first one
)

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

	proc := newProc(conf, p, logger.Clone().SetPrefix(*name))

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
		conf.CopyEnv,
	)
}
