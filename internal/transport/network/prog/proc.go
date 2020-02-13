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

const maxCache = 10000 // Max size for caches before lines are dropped

func newProc(conf *network.Config, p *process.Process, l *log.Logger) *Proc {
	return &Proc{
		process:    p,
		conf:       conf,
		log:        l,
		doneChan:   make(chan struct{}),
		stdioCond:  sync.NewCond(&sync.Mutex{}),
		stdIOLines: make([]protocol.StdIOLine, 0, maxCache),
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

	stdioCond  *sync.Cond
	stdIOLines []protocol.StdIOLine
	stdioSeq   int64

	// TODO: Disconnection Message (eg if we're disconnected from the server on the other side)
	// TODO: rehash/reset config method

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

	p.stdIOLines = p.stdIOLines[:0]
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

func makeCopy(src []protocol.StdIOLine) (out []protocol.StdIOLine) {
	out = make([]protocol.StdIOLine, len(src))
	copy(out, src)
	return out
}

func findSeq(target int64, slice []protocol.StdIOLine) int {
	for i, v := range slice {
		if v.ID == target {
			return i
		}
	}

	return -1
}

func getAllAfter(seq int64, slice []protocol.StdIOLine) []protocol.StdIOLine {
	idx := findSeq(seq, slice)
	if seq == -1 || idx == -1 {
		return makeCopy(slice)
	}
	if len(slice) > idx {
		return makeCopy(slice[idx+1:])
	}
	return []protocol.StdIOLine{}
}

// GetStdioLines gets the current buffered stdio lines. The lastSeen arg is the place from which to start
// the returned lines slice, if it less than zero, all buffered lines are sent.
// Otherwise, if lastSeen nonzero, and there are no new lines, GetStdout blocks until at least one new line is seen.
//
// This method is an RPC method, which is why no error is returned but it is still marked as
// returning one.
func (p *Proc) GetStdioLines(lastSeq int64, out *protocol.StdIOLines) error { //nolint:unparam // sig required by rpc
	p.stdioCond.L.Lock()
	defer p.stdioCond.L.Unlock()

	var err error
	if !p.process.IsRunning() {
		// not returning here so that residual lines can be accessed
		err = errors.New("not running")
	}

	if lastSeq < 0 && len(p.stdIOLines) > 0 {
		*out = protocol.StdIOLines{
			Lines: makeCopy(p.stdIOLines),
			Error: protocol.SErrorFromError(err),
		}
		return nil
	} else if !p.process.IsRunning() {
		// We have a last seen, BUT, we're also not expecting any more lines. Therefore, return what we have
		*out = protocol.StdIOLines{
			Lines: getAllAfter(lastSeq, p.stdIOLines),
			Error: protocol.SErrorFromError(err),
		}
		return nil
	}

	// Loop as suggested by docs: https://golang.org/pkg/sync/#Cond.Wait
	for {
		p.stdioCond.Wait()
		if !p.process.IsRunning() {
			err = errors.New("not running")
			break
		} else if len(p.stdIOLines) > 0 && p.stdIOLines[len(p.stdIOLines)-1].ID != lastSeq {
			break
		}
	}

	*out = protocol.StdIOLines{
		Lines: getAllAfter(lastSeq, p.stdIOLines),
		Error: protocol.SErrorFromError(err),
	}

	return nil
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
		p.stdioCond.Broadcast()
	}
	if err := s.Err(); err != nil {
		p.log.Warn(err)
	}
	<-p.doneChan
	p.stdioCond.Broadcast()
}

func (p *Proc) cacheLine(line string, stdout bool) {
	p.stdioCond.L.Lock()
	defer p.stdioCond.L.Unlock()
	p.stdioSeq++
	if len(p.stdIOLines) > maxCache {
		p.stdIOLines = p.stdIOLines[len(p.stdIOLines)-maxCache:]
	}

	p.stdIOLines = append(p.stdIOLines, protocol.StdIOLine{Line: line, Stdout: stdout, ID: p.stdioSeq})
}
