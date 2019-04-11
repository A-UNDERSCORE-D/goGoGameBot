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

	"github.com/dustin/go-humanize"
	"github.com/shirou/gopsutil/process"
	"golang.org/x/sys/unix"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

func NewProcess(command string, args []string, workingDir string, logger *log.Logger) (*Process, error) {

	p := &Process{
		commandString: command,
		argListString: args,
		workingDir:    workingDir,
		StdinMutex:    sync.Mutex{},
		log:           logger,
	}
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
	commandMutex  sync.Mutex
	Stderr        io.ReadCloser
	Stdout        io.ReadCloser
	Stdin         io.WriteCloser
	StdinMutex    sync.Mutex
	DoneChan      chan bool
	log           *log.Logger
	statusMutex   sync.Mutex
	hasStarted    bool
	hasExited     bool
}

// UpdateCmd sets the command and arguments to be used when creating the exec.Cmd used internally.
// It is safe for concurrent use
func (p *Process) UpdateCmd(command string, args []string, workingDir string) {
	p.commandMutex.Lock()
	defer p.commandMutex.Unlock()
	p.commandString = command
	p.argListString = args
	p.workingDir = workingDir
}

func (p *Process) setupCmd() error {
	p.commandMutex.Lock()
	cmd := exec.Command(p.commandString, p.argListString...)
	cmd.Dir = p.workingDir
	p.commandMutex.Unlock()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	p.cmd = cmd
	p.Stdin = stdin
	p.Stdout = stdout
	p.Stderr = stderr
	return nil
}

func (p *Process) Reset() error {
	p.DoneChan = make(chan bool) // TODO: This causes a data race. In theory it's not an issue, but it'd be nice to fix it
	p.statusMutex.Lock()
	p.hasStarted = false
	p.hasExited = false
	p.statusMutex.Unlock()
	return p.setupCmd()
}

func (p *Process) Start() error {
	p.log.Info("Starting")
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("could not start process: %v", err)
	}

	p.statusMutex.Lock()
	p.hasStarted = true
	p.statusMutex.Unlock()
	return nil
}

func (p *Process) IsRunning() bool {
	p.statusMutex.Lock()
	defer p.statusMutex.Unlock()
	return p.hasStarted && !p.hasExited
}

func (p *Process) GetReturnStatus() string {
	return p.cmd.ProcessState.String()
}

func (p *Process) GetReturnCode() int {
	return p.cmd.ProcessState.ExitCode()
}

func (p *Process) GetStatus() string {
	out := strings.Builder{}
	if !p.IsRunning() {
		out.WriteString("$c[red]$b Not running$r")
		return out.String()
	}
	ps, err := process.NewProcess(int32(p.cmd.Process.Pid))
	if err != nil {
		return fmt.Sprintf("$b$c[red]ERROR:$b %s", err)
	}
	out.WriteString("$c[light green]$bRunning$r: ")
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

// writes data to stdin on this process, adding a newline if one does not exist
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

func (p *Process) WriteString(toWrite string) (int, error) {
	return p.Write([]byte(toWrite))
}

func (p *Process) WaitForCompletion() error {
	defer close(p.DoneChan)
	err := p.cmd.Wait()
	p.statusMutex.Lock()
	p.hasExited = true
	p.statusMutex.Unlock()
	if err != nil {
		return err
	}
	return nil
}

// sends the given signal to the underlying process
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

// sends SIGTERM to the process if it is running
func (p *Process) Stop() error {
	return p.SendSignal(unix.SIGTERM)
}

// sends SIGKILL to the process if it is running
func (p *Process) Kill() error {
	return p.SendSignal(unix.SIGKILL)
}

// asks the process to stop and waits for the configured timeout, after which it kills the process
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
		return p.Kill()

	case <-p.DoneChan:
		return nil
	}
}
