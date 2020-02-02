package main

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/process"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/network"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/network/protocol"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

func newProc(conf *network.Config, p *process.Process, l *log.Logger) *Proc {
	return &Proc{
		process:    p,
		conf:       conf,
		log:        l,
		stdoutCond: sync.NewCond(&sync.Mutex{}),
		stderrCond: sync.NewCond(&sync.Mutex{}),
		doneChan:   make(chan struct{}),
	}
}

// Nothing is just that, Nothing. Its used to stub out arguments or "returns" in RPC calls that have neither
type Nothing struct{}

// TODO: something to allow Proc to send status messages back, similar to the existing stdio stuff

// Proc is a net/rpc compatible process runner for use with the Network transport in GGGB
type Proc struct {
	process *process.Process
	conf    *network.Config
	log     *log.Logger

	// caches for stdout/err
	stdout     []string
	stdoutCond *sync.Cond
	stderr     []string
	stderrCond *sync.Cond

	// TODO: Disconnection Message (eg if we're disconnected from the server on the other side)
	// TODO: rehash/reset config method
	// TODO: message IDs of some sort, for when a message is repeated often

	doneChan chan struct{}
}

func getListener(conf *network.Config) (net.Listener, error) {
	networkType := "tcp"
	if conf.IsUnix {
		networkType = "unix"
	}

	listener, err := net.Listen(networkType, conf.Address)
	if err != nil {
		if strings.HasSuffix(err.Error(), "bind: address already in use") && conf.IsUnix {
			os.Remove(conf.Address)
			return getListener(conf)
		}
		return nil, err
	}
	return listener, nil
}

// Start starts the process and instantly returns
func (p *Proc) Start(_ Nothing, outErr *protocol.SerialiseError) error { //nolint:unparam // Sig as required for rpc
	if err := p.reset(); err != nil {
		*outErr = protocol.SErrorFromError(err)
		return nil
	}

	go p.monitorStdio(true)
	go p.monitorStdio(false)
	err := p.process.Start()
	if err != nil {
		*outErr = protocol.SErrorFromError(err)
		return nil
	}

	go func() {
		if err := p.process.WaitForCompletion(); err != nil {
			p.log.Warnf("error occurred with process: %s", err)
		}
		close(p.doneChan)
	}()
	return nil
}

func (p *Proc) reset() error {
	if err := p.process.Reset(); err != nil {
		return err
	}

	p.stdout = p.stdout[:0]
	p.stderr = p.stderr[:0]
	p.doneChan = make(chan struct{})
	return nil
}

// StopOrKillTimeout stops or kills the running process, waiting the specified timeout before killing the process
func (p *Proc) StopOrKillTimeout(timeout time.Duration, out *protocol.SerialiseError) error { //nolint:unparam // RPC
	if err := p.process.StopOrKillTimeout(timeout); err != nil {
		*out = protocol.SErrorFromError(err)
	}
	return nil
}

func findReverse(slice []string, target string) int {
	for i := len(slice) - 1; i >= 0; i-- {
		if slice[i] == target {
			return i
		}
	}

	return -1
}

func makeCopy(src []string) (out []string) {
	out = make([]string, len(src))
	copy(out, src)
	return out
}

func getLinesAfter(target string, slice []string) []string {
	idx := findReverse(slice, target)
	if idx == -1 {
		return makeCopy(slice)
	}
	if len(slice) > idx {
		return makeCopy(slice[idx+1:])
	}
	return []string{}
}

// GetStdout gets the current buffered stdout lines. The lastSeen arg is the place from which to start
// the returned lines slice, if it is omitted (== "") then all buffered lines are sent.
// Otherwise, if lastSeen is not empty, and there are no new lines, GetStdout blocks until
// at least one is seen.
// This method is an RPC method, which is why no error is returned but it is still marked as
// returning one.
func (p *Proc) GetStdout(lastSeen string, out *protocol.StdIOLines) error { //nolint:unparam // sig required by rpc
	res, err := p.getStd(true, lastSeen)
	*out = protocol.StdIOLines{Lines: res, Error: protocol.SErrorFromError(err), Stdout: true}
	return nil
}

// GetStderr is like GetStdout but for stderr
func (p *Proc) GetStderr(lastSeen string, out *protocol.StdIOLines) error { //nolint:unparam // sig required by rpc
	res, err := p.getStd(false, lastSeen)
	*out = protocol.StdIOLines{Lines: res, Error: protocol.SErrorFromError(err), Stdout: false}
	return nil
}

func (p *Proc) getStd(stdout bool, lastSeen string) (outSlice []string, err error) {
	if !p.process.IsRunning() {
		// not returning here so that residual lines can be accessed
		err = errors.New("not running")
	}

	cond := p.stdoutCond
	slice := &p.stdout
	if !stdout {
		cond = p.stderrCond
		slice = &p.stderr
	}
	cond.L.Lock()
	defer cond.L.Unlock()

	if lastSeen == "" && len(*slice) > 0 {
		return makeCopy(*slice), nil
	} else if !p.process.IsRunning() {
		// We have a last seen, BUT, we're also not expecting any more lines. Therefore, return what we have
		return getLinesAfter(lastSeen, *slice), err
	}

	// Loop as suggested by docs: https://golang.org/pkg/sync/#Cond.Wait
	for {
		cond.Wait()
		// Have we added new lines to the cache? not in the for clause as cond.Wait() messes with some of our locks
		if !p.process.IsRunning() || len(*slice) > 0 && (*slice)[len(*slice)-1] != lastSeen {
			break
		}
	}

	// okay we have new lines. Lets send them back
	return getLinesAfter(lastSeen, *slice), err
}

// GetStatus returns the current state of the transport
func (p *Proc) GetStatus(_ Nothing, out *util.TransportStatus) error { //nolint:unparam // signature required by rpc
	if p.process.IsRunning() {
		*out = util.Running
	} else {
		*out = util.Stopped
	}
	return nil
}

// GetHumanStatus returns the status of the transport in a human readable form
func (p *Proc) GetHumanStatus(_ Nothing, out *string) error { //nolint:unparam // signature required by rpc
	*out = p.process.GetStatus()
	return nil
}

func (p *Proc) Write(toWrite []byte, out *protocol.SerialiseError) error { //nolint:unparam // signature required by rpc
	_, err := p.process.Write(toWrite)
	*out = protocol.SErrorFromError(err)
	return nil
}

// Wait waits for the process to exit
func (p *Proc) Wait(_ Nothing, res *protocol.ProcessExit) error { //nolint:unparam // signature required by rpc
	if !p.process.IsRunning() {
		*res = protocol.ProcessExit{Error: protocol.SErrorFromString("not running")}
		return nil
	}

	<-p.doneChan
	*res = protocol.ProcessExit{
		Return:    p.process.GetReturnCode(),
		StrReturn: p.process.GetReturnStatus(),
		Error:     protocol.SerialiseError{},
	}
	return nil
}

// Ping takes a time as an arg and instantly returns it. This allows for relatively accurate latency monitoring
func (p *Proc) Ping(t time.Time, res *time.Time) error {
	*res = t
	return nil
}

const (
	stdoutStr = "STDOUT"
	stderrStr = "STDERR"
)

func (p *Proc) monitorStdio(stdout bool) {
	f := p.process.Stdout
	prefix := stdoutStr
	if !stdout {
		f = p.process.Stderr
		prefix = stderrStr
	}

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		p.log.Infof("[%s] %s", prefix, line)
		p.cacheLine(line, stdout)

		// Tell our waiters, if any, that there are new lines
		if stdout {
			p.stdoutCond.Broadcast()
		} else {
			p.stderrCond.Broadcast()
		}
	}
	if err := s.Err(); err != nil {
		p.log.Warn(err)
	}
	<-p.doneChan
	if stdout {
		p.stdoutCond.Broadcast()
	} else {
		p.stderrCond.Broadcast()
	}
}

func (p *Proc) cacheLine(line string, stdout bool) {
	if stdout {
		p.stdoutCond.L.Lock()
		defer p.stdoutCond.L.Unlock()
		if len(p.stdout) > maxCache {
			p.stdout = p.stdout[len(p.stdout)-maxCache:]
		}

		p.stdout = append(p.stdout, line)
	} else {
		p.stderrCond.L.Lock()
		defer p.stderrCond.L.Unlock()
		if len(p.stderr) > maxCache {
			p.stderr = p.stderr[len(p.stdout)-maxCache:]
		}

		p.stderr = append(p.stderr, line)
	}
}
