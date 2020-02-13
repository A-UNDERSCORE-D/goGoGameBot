// ...
package processTransport

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/anmitsu/go-shlex"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/mutexTypes"
)

func New(transportConfig config.TransportConfig, logger *log.Logger) (*ProcessTransport, error) {
	p := ProcessTransport{log: logger.SetPrefix(logger.Prefix() + "|" + "PT")}
	if err := p.Update(transportConfig); err != nil {
		return nil, err
	}
	return &p, nil
}

// ProcessTransport is a transport implementation that works with a process.Process to
// provide local-to-us game servers
type ProcessTransport struct {
	process *process.Process
	log     *log.Logger
	stdout  chan []byte
	stderr  chan []byte

	status mutexTypes.Int
}

// GetStatus returns the current state of the game the transport manages
func (p *ProcessTransport) GetStatus() util.TransportStatus {
	if p.process.IsRunning() {
		return util.Running
	}
	return util.Stopped
}

// GetHumanStatus returns the status of the transport that is human readable
func (p *ProcessTransport) GetHumanStatus() string {
	return p.process.GetStatus()
}

func (p *ProcessTransport) monitorStdIO() error {
	if !p.process.IsRunning() {
		return errors.New("cannot watch stdio on a non-running game")
	}
	go func() {
		s := bufio.NewScanner(p.process.Stdout)
		last := ""
		for s.Scan() {
			b := s.Bytes()
			p.getStdioChan(true) <- b
			last = string(b)
		}
		close(p.getStdioChan(true))
		p.log.Infof("stdout exit: %q", last)
	}()

	go func() {
		s := bufio.NewScanner(p.process.Stderr)
		for s.Scan() {
			p.getStdioChan(false) <- s.Bytes()
		}
		close(p.getStdioChan(false))
		p.log.Info("stderr exit")
	}()

	return nil
}

func (p *ProcessTransport) getStdioChan(stdout bool) chan []byte {
	if stdout {
		if p.stdout == nil {
			p.stdout = make(chan []byte)
		}
		return p.stdout
	}
	if p.stderr == nil {
		p.stderr = make(chan []byte)
	}
	return p.stderr
}

// Stdout returns a channel that will have lines from stdout sent over it.
func (p *ProcessTransport) Stdout() <-chan []byte {
	return p.getStdioChan(true)
}

// Stderr returns a channel that will have lines from stderr sent over it
func (p *ProcessTransport) Stderr() <-chan []byte {
	return p.getStdioChan(false)
}

// Update updates the Transport with a TransportConfig
func (p *ProcessTransport) Update(rawConf config.TransportConfig) error {
	conf := new(Config)
	if err := xml.Unmarshal([]byte(rawConf.Config), conf); err != nil {
		return err
	}

	wdir := conf.WorkingDirectory
	if wdir == "" {
		wdir = path.Dir(conf.Path)
		p.log.Infof("working directory inferred to %q from binary path %q", wdir, conf.Path)
	}

	procArgs, err := shlex.Split(conf.Args, true)
	if err != nil {
		return fmt.Errorf("could not parse arguments: %w", err)
	}
	if p.process == nil {
		l := p.log.Clone().SetPrefix(p.log.Prefix() + "|" + "P")
		proc, err := process.NewProcess(conf.Path, procArgs, wdir, l, conf.Environment, !conf.DontCopyEnv)
		if err != nil {
			return err
		}
		p.process = proc
	} else {
		p.process.UpdateCmd(conf.Path, procArgs, wdir, conf.Environment, !conf.DontCopyEnv)
	}
	return nil
}

func (p *ProcessTransport) StopOrKill() error {
	return p.StopOrKillTimeout(time.Second * 30)
}

func (p *ProcessTransport) StopOrKillTimeout(duration time.Duration) error {
	if !p.IsRunning() {
		return util.ErrorNotRunning
	}
	return p.process.StopOrKillTimeout(duration)
}

func (p *ProcessTransport) StopOrKillWaitgroup(group *sync.WaitGroup) {
	group.Add(1)
	if err := p.StopOrKill(); err != nil {
		p.log.Warnf("error while stopping game: %s", err)
	}
	group.Done()
}

// Run runs the process once, if it is not already running. It blocks until the process exits
func (p *ProcessTransport) Run(start chan struct{}) (int, string, error) {
	closed := false

	defer func() {
		if !closed {
			close(start)
		}
	}()

	if p.IsRunning() {
		return -1, "", fmt.Errorf("could not start game: %w", util.ErrorAlreadyRunning)
	}

	if err := p.process.Reset(); err != nil {
		return -1, "", fmt.Errorf("could not reset process: %w", err)
	}

	p.stdout = nil
	p.stderr = nil
	if err := p.process.Start(); err != nil {
		return -1, "", fmt.Errorf("could not start process: %w", err)
	}

	close(start)
	closed = true

	if err := p.monitorStdIO(); err != nil {
		go p.StopOrKill()
		return -1, "", fmt.Errorf("could not begin monitoring standard i/o. Aborting: %w", err)
	}

	if err := p.process.WaitForCompletion(); err != nil && !errors.Is(err, &exec.ExitError{}) {
		return p.process.GetReturnCode(), p.process.GetReturnStatus(), err
	}

	return p.process.GetReturnCode(), p.process.GetReturnStatus(), nil
}

func (p *ProcessTransport) IsRunning() bool {
	return p.process.IsRunning()
}

func (p *ProcessTransport) Write(b []byte) (n int, err error) {
	return p.process.Write(b)
}

func (p *ProcessTransport) WriteString(s string) (n int, err error) {
	return p.process.WriteString(s)
}
