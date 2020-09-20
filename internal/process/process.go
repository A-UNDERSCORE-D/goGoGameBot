// Package process holds process management code for direct interaction with processes
package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize" // nolint:misspell // I dont control others' package names
	psutilProc "github.com/shirou/gopsutil/process"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/mutexTypes"
)

// NewProcess returns a ready to use process object with the given options. If any errors occur during creation and
// setup, they are returned
func NewProcess(cmd string, args []string, workingDir string, logger *log.Logger, env []string, copySystemEnv bool) (*Process, error) { //nolint:lll // unavoidable
	p := &Process{
		log: logger,
	}
	p.UpdateCmd(cmd, args, workingDir, env, copySystemEnv)

	if err := p.Reset(); err != nil {
		return nil, err
	}

	return p, nil
}

// Process is a representation of a command to be run and access to its stdin/out/err
type Process struct {
	cmd           *exec.Cmd
	commandString string
	argListString []string
	workingDir    string
	cmdEnv        []string
	commandMutex  sync.Mutex
	stdioWg       *sync.WaitGroup
	Stderr        io.Reader
	Stdout        io.Reader
	Stdin         io.WriteCloser
	StdinMutex    sync.Mutex
	DoneChan      chan bool
	log           *log.Logger
	hasStarted    mutexTypes.Bool
	hasExited     mutexTypes.Bool
}

func getEnv(baseEnvs []string, copySystemEnv bool) []string {
	var envs []string
	if copySystemEnv {
		envs = append(envs, os.Environ()...)
	}

	envs = append(envs, baseEnvs...) // Do this second for overriding

	return envs
}

// UpdateCmd sets the command and arguments to be used when creating the exec.Cmd used internally.
// It is safe for concurrent use. Note that this will only take effect on the next reset of the Process object
func (p *Process) UpdateCmd(command string, args []string, workingDir string, env []string, copySystemEnv bool) {
	p.commandMutex.Lock()
	defer p.commandMutex.Unlock()
	p.commandString = command
	p.argListString = args
	p.workingDir = workingDir
	p.cmdEnv = getEnv(env, copySystemEnv)
}

func (p *Process) setupCmd() error {
	p.commandMutex.Lock()

	cmd := exec.Command(p.commandString, p.argListString...) //nolint:gosec // its intentional
	cmd.Dir = p.workingDir
	cmd.Env = p.cmdEnv

	p.commandMutex.Unlock()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stdErr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	p.stdioWg = new(sync.WaitGroup)
	p.stdioWg.Add(2)

	p.cmd = cmd // TODO: racy on really quick restarts?
	p.Stdin = stdin
	p.Stdout = waitGroupIoCopy(p.stdioWg, stdOut)
	p.Stderr = waitGroupIoCopy(p.stdioWg, stdErr)

	return nil
}

// Reset cleans up an already-run process, making it ready to be run again
func (p *Process) Reset() error {
	p.DoneChan = make(chan bool) // TODO: This causes a data race. In theory it's not an issue, but it'd be nice to fix it
	p.hasStarted.Set(false)
	p.hasExited.Set(false)

	return p.setupCmd()
}

// Start starts the process, if startup errors, that error is returned
func (p *Process) Start() error {
	p.log.Info("Starting")

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("could not start process: %v", err)
	}

	p.hasStarted.Set(true)

	return nil
}

// IsRunning returns whether or not the current process is running
func (p *Process) IsRunning() bool {
	return p.hasStarted.Get() && !p.hasExited.Get()
}

// GetReturnStatus returns a string containing the return status of the process, an example would be "exit code 1"
func (p *Process) GetReturnStatus() string {
	return p.cmd.ProcessState.String()
}

// GetReturnCode returns the exit code of the process as an int. Calling this on a Process that has not been started
// will panic
func (p *Process) GetReturnCode() int {
	return p.cmd.ProcessState.ExitCode()
}

// GetStatus returns the current status of the process, including memory and CPU usage
func (p *Process) GetStatus() string {
	out := strings.Builder{}

	if !p.IsRunning() {
		out.WriteString("$cFF0000$bNot running$r")
		return out.String()
	}

	ps, err := psutilProc.NewProcess(int32(p.cmd.Process.Pid))

	if err != nil {
		return fmt.Sprintf("$b$cFF0000ERROR:$b %s", err)
	}

	out.WriteString("$c00FC00$bRunning$r: ")
	out.WriteString("CPU usage: ")

	cpuPercentage, err := ps.CPUPercent()
	if err != nil {
		out.WriteString("Error ")
	} else {
		out.WriteString(fmt.Sprintf("%.2f%% ", cpuPercentage))
	}

	out.WriteString("Memory Usage: ")

	if m, err := ps.MemoryInfo(); err != nil {
		out.WriteString("Error: %s ")
	} else {
		out.WriteString(humanize.IBytes(m.RSS))
	}

	if m, err := ps.MemoryPercent(); err == nil {
		out.WriteString(fmt.Sprintf(" (%.2f%%)", m))
	}

	return out.String()
}

// Write writes data to stdin on this process, adding a newline if one does not exist
func (p *Process) Write(data []byte) (int, error) {
	toWrite := data
	if !bytes.HasSuffix(toWrite, []byte{'\n'}) {
		toWrite = append(toWrite, '\n')
	}

	p.StdinMutex.Lock()
	defer p.StdinMutex.Unlock()
	p.log.Infof("[STDIN] %s", data)

	return p.Stdin.Write(toWrite)
}

// WriteString writes the given string to the stdin of the running process, If a newline does not end the given string.
// it is added
func (p *Process) WriteString(toWrite string) (int, error) {
	return p.Write([]byte(toWrite))
}

// WaitForCompletion blocks until the Process's command has completed. If an error occurs while waiting, it is returned
func (p *Process) WaitForCompletion() error {
	defer close(p.DoneChan)
	p.stdioWg.Wait()
	err := p.cmd.Wait()
	p.hasExited.Set(true)

	if err != nil {
		return err
	}

	return nil
}

// SendSignal sends the given signal to the underlying process
func (p *Process) SendSignal(sig os.Signal) error {
	if !p.IsRunning() {
		return errors.New("attempt to send non-running process a signal")
	}

	if err := p.cmd.Process.Signal(sig); err != nil {
		p.log.Warnf("could not send signal %s to process: %s", sig.String(), err)
		return err
	}

	return nil
}

// Stop sends SIGTERM to the process if it is running
func (p *Process) Stop() error {
	return p.SendSignal(os.Interrupt)
}

// Kill sends SIGKILL to the process if it is running
func (p *Process) Kill() error {
	return p.SendSignal(os.Kill)
}

// StopOrKillTimeout asks the process to stop and waits for the configured timeout, after which it kills the process
func (p *Process) StopOrKillTimeout(timeout time.Duration) error {
	if !p.IsRunning() {
		return nil
	}

	err := p.Stop()
	if err != nil {
		return err
	}

	select {
	case <-time.After(timeout):
		p.log.Infof("killing process as %s has elapsed without a polite exit", timeout)
		return p.Kill()

	case <-p.DoneChan:
		return nil
	}
}
