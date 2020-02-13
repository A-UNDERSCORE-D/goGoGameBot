package network

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/rpc"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/network/protocol"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/mutexTypes"
)

var (
	debugRPC = false
)

// New creates a new network Transport
func New(conf config.TransportConfig, logger *log.Logger) (*Transport, error) {
	t := &Transport{
		logger:  logger.Clone().SetPrefix("net"),
		stdout:  make(chan []byte),
		stderr:  make(chan []byte),
		done:    make(chan struct{}),
		lastSeq: -1,
	}
	if err := t.Update(conf); err != nil {
		return nil, err
	}

	return t, nil
}

var nothing = struct{}{}

// TODO: How do we deal with a closed client? we need to detect closure and autoreconnect or similar

// Transport is an implementation of the transport interface that is designed to be used over the network
type Transport struct {
	logger *log.Logger
	stdout chan []byte
	stderr chan []byte
	*Config
	startingLocal bool // are we currently attempting to exec it?
	client        *rpc.Client
	done          chan struct{}
	isConnected   mutexTypes.Bool

	pings []time.Duration
}

func formatRPCCall(target, methodName string, args, res interface{}) string {
	return fmt.Sprintf("%s.%s(%#v) = %#v", target, methodName, args, res)
}

func (t *Transport) call(methodName string, arg, res interface{}) error {
	if t.client == nil {
		t.logger.Warnf(
			"attempt to make RPC call with nil transport client: %s",
			formatRPCCall(protocol.RPCName, methodName, arg, res),
		)
		t.logger.Debug(string(debug.Stack()))
		return errors.New("transport client is nil, cannot make call")
	}
	if debugRPC {
		t.logger.Debugf("Making RPC Call: %s", formatRPCCall(protocol.RPCName, methodName, arg, res))
	}

	err := t.client.Call(protocol.RPCName+"."+methodName, arg, res)
	if err != nil {
		t.logger.Warnf("got an error from client.Call: %s", err)
		t.isConnected.Set(false) // TODO: does this need more?
	}
	return err
}

// callGo is like call, it wraps the client Go method for status monitoring of the Client. In the case of an error,
// it is handled gracefully
func (t *Transport) callGo(methodName string, arg, res interface{}) (<-chan *rpc.Call, error) {
	if t.client == nil {
		t.logger.Warnf(
			"attempt to make RPC call with nil transport client: %s",
			formatRPCCall(protocol.RPCName, methodName, arg, res),
		)
		t.logger.Debug(debug.Stack())
		return nil, errors.New("transport client is nil, cannot make call")
	}

	if debugRPC {
		t.logger.Debugf("Making async RPC Call: %s", formatRPCCall(protocol.RPCName, methodName, arg, res))
	}

	// make the Go call, start our own goroutine to wrap it, check the error before passing the value outwards
	internalChan := make(chan *rpc.Call, 1)
	outChan := make(chan *rpc.Call, 1)
	call := t.client.Go(protocol.RPCName+"."+methodName, arg, res, internalChan)
	go func() {
		goRes := <-call.Done
		if goRes.Error != nil {
			t.logger.Warnf("got an error from client.Call: %s", goRes.Error)
			t.isConnected.Set(false) // TODO: does this need more?
		}
		outChan <- goRes
	}()

	return outChan, nil
}

// GetStatus returns the current state of the transport
func (t *Transport) GetStatus() util.TransportStatus {
	res := new(util.TransportStatus)
	if err := t.call("GetStatus", nothing, res); err != nil {
		t.logger.Warn("asd", err)
		return util.Unknown
	}
	t.logger.Infof("sd %#v", res)
	return *res
}

// GetHumanStatus returns the status of the transport that is human readable
func (t *Transport) GetHumanStatus() string {
	panic("not implemented")
}

func (t *Transport) getStdioChan(stdout bool) chan []byte {
	if stdout {
		if t.stdout == nil {
			t.stdout = make(chan []byte)
		}
		return t.stdout
	}
	if t.stderr == nil {
		t.stderr = make(chan []byte)
	}
	return t.stderr
}

// Stdout returns a channel that will have lines from stdout sent over it.
// multiple calls are not supported.
func (t *Transport) Stdout() <-chan []byte {
	return t.getStdioChan(true)
}

// Stderr returns a channel that will have lines from stdout sent over it.
// multiple calls are not supported.
func (t *Transport) Stderr() <-chan []byte {
	return t.getStdioChan(false)
}

// Update updates the Transport with a TransportConfig
func (t *Transport) Update(confHolder config.TransportConfig) error {
	conf := new(Config)
	if err := xml.Unmarshal([]byte(confHolder.Config), conf); err != nil {
		return err
	}
	t.Config = conf
	return nil
}

// StopOrKill implements StopOrKiller
func (t *Transport) StopOrKill() error {
	return t.StopOrKillTimeout(time.Second * 30)
}

// StopOrKillTimeout implements StopOrKiller
func (t *Transport) StopOrKillTimeout(duration time.Duration) error {
	res := new(protocol.SerialiseError)
	if err := t.call("StopOrKillTimeout", duration, res); err != nil {
		return err
	}
	if res.IsError {
		return res.ToError()
	}
	return nil
}

// StopOrKillWaitgroup implements StopOrKiller
func (t *Transport) StopOrKillWaitgroup(group *sync.WaitGroup) {
	group.Add(1)

	if err := t.StopOrKill(); err != nil {
		t.logger.Warnf("error occurred while attempting to stop: %s", err)
	}

	group.Done()
}

func (t *Transport) dialOrStart(typ, address string) (*rpc.Client, error) {
	client, err := rpc.Dial(typ, address)
	if err != nil && !t.StartLocal || t.startingLocal {
		return nil, err
	} else if err == nil {
		// okay we're done, it worked
		return client, nil
	}

	// Start a Proc instance. This assumes there's a compiled version in our working directory
	// TODO: allow the config to specify how to go about this. With the requirement that we can tack on args as we want
	cmd := exec.Command("./proc")

	// Set this to its own process group -- prevents killing it if we're ^C-ed
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	t.startingLocal = true

	defer func() { t.StartLocal = false }()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	time.Sleep(time.Microsecond * 50)
	return t.dialOrStart(typ, address)

}

func (t *Transport) start() error {
	typ := "tcp"
	if t.IsUnix {
		typ = "unix"
	}

	client, err := t.dialOrStart(typ, t.Address)
	if err != nil {
		return err
	}

	t.isConnected.Set(true)
	go t.monitorLatency()

	t.client = client
	outError := new(protocol.SerialiseError)
	if err := t.call("Start", nothing, outError); err != nil {
		return fmt.Errorf("could not make call: %w", err)
	}

	if outError.IsError {
		return fmt.Errorf("could not start process: %w", outError.ToError())
	}
	return nil
}

// Run runs the underlying process on the Transport. It returns the return code of the process (or -1 if start failed)
// a string representation of the exit, if applicable, and an error. error should be checked first as the string
// may not be filled for some errors.
func (t *Transport) Run(start chan struct{}) (retcode int, ret string, _ error) {
	// TODO: try and connect when we are created, or on any call
	closed := false

	defer func() {
		if !closed {
			close(start)
		}
	}()

	// TODO: either ensure that start behaves well when the other side is already running, or otherwise deal with that

	if err := t.start(); err != nil {
		return -1, "", fmt.Errorf("could not start game: %w", err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	stdioCtx, stdioCancel := context.WithCancel(context.Background())

	defer func() {
		stdioCancel()
		wg.Wait()
	}()

	if err := t.monitorStdio(stdioCtx, wg); err != nil {
		return -1, "", err
	}

	closed = true
	close(start)

	res := protocol.ProcessExit{}
	returned := false

	doneChan, err := t.callGo("Wait", nothing, &res)
	if err != nil {
		return -1, "", fmt.Errorf("could not call Wait on game: %w", err)
	}

	select {
	case call := <-doneChan:
		returned = true
		if err := call.Error; err != nil {
			t.logger.Warnf("error occurred while waiting: %s", err)
		}
	case <-t.done:
		// we were asked to stop externally
	}

	if returned {
		return res.Return, res.StrReturn, res.Error.ToError()
	}

	return -1, "", nil
}

func (t *Transport) monitorStdio(ctx context.Context, wg *sync.WaitGroup) error {
	if !t.IsRunning() {
		return errors.New("cannot monitor stdio: Not running")
	}

	go func() {
		defer func() {
			close(t.stdout)
			close(t.stderr)
			t.stdout = nil
			t.stderr = nil
			wg.Done()
		}()
		for {
			err := t.getAndHandleStdLines(ctx, t.lastSeq)
			if err != nil && errors.Is(err, context.Canceled) {
				break
			} else if err != nil && err.Error() != "not running" {
				t.logger.Warnf("error from t.getAndHandleStdLines: %s", err)
			}
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := t.getAndHandleStdLines(timeoutCtx, t.lastSeq)

		if errors.Is(context.DeadlineExceeded, err) {
			return
		} else if err != nil && err.Error() != "not running" {
			t.logger.Warnf("got an error from t.GetStdLines: %s", err)
		}

	}()
	return nil
}

func (t *Transport) getAndHandleStdLines(ctx context.Context, lastSeq int64) error {
	lines, err := t.getStdLines(ctx, lastSeq)
	t.handleStdioLines(lines)
	return err
}

func (t *Transport) getStdLines(ctx context.Context, lastSeq int64) (*protocol.StdIOLines, error) {
	lines := new(protocol.StdIOLines)
	callDone, err := t.callGo("GetStdioLines", lastSeq, lines)
	if err != nil {
		return lines, err
	}
	select {
	case <-ctx.Done():
		return lines, ctx.Err()
	case <-callDone:
		return lines, lines.Error.ToError()
	}
}

func (t *Transport) handleStdioLines(lines *protocol.StdIOLines) {
	var c chan []byte
	defer func() {
		if err := recover(); err != nil && err == "send on closed channel" {
			// TODO: there a better way to do this? maybe with a context? (cc processTransport)
			// TODO: Actually I think just doing a nonblocking send would be better for this. If you dont want more
			// TODO: Lines, then just dont listen. I dont think I ever do that in game but I'd rather support it than
			// TODO: not. And managing a possibly closed channel from the sender's side is a mess.

			t.logger.Tracef("caught send on closed channel panic in networkTransport.handleStdioLines")
			if c == t.stdout {
				t.stdout = nil
			} else {
				t.stderr = nil
			}
		} else if err != nil {
			panic(err)
		}
	}()

	for _, v := range lines.Lines {
		c = t.getStdioChan(v.Stdout)
		c <- []byte(v.Line)
		t.lastSeq = v.ID
	}
}

// IsRunning returns whether or not the underlying process is currently running. For more information use GetStatus.
func (t *Transport) IsRunning() bool {
	return t.GetStatus() == util.Running // TODO: add a connected check here?
}

func (t *Transport) Write(p []byte) (n int, err error) {
	outErr := new(protocol.SerialiseError)
	if err := t.call("Write", p, outErr); err != nil {
		return -1, err
	}
	if outErr.IsError {
		return -1, outErr.ToError()
	}
	return len(p), nil
}

// WriteString implements Transport
func (t *Transport) WriteString(s string) (n int, err error) {
	return t.Write([]byte(s))
}

func (t *Transport) monitorLatency() {
	for t.isConnected.Get() {
		resTime := new(time.Time)
		if err := t.call("Ping", time.Now(), resTime); err != nil {
			t.logger.Warnf("error while attempting to get ping time: %s", err)
		}
		dur := time.Since(*resTime)
		t.pings = append(t.pings, dur)
		time.Sleep(time.Second * 5)
	}
}
