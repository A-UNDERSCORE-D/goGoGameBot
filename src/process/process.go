package process

import (
    "fmt"
    "golang.org/x/sys/unix"
    "io"
    "log"
    "os"
    "os/exec"
    "strings"
    "sync"
    "time"
)

func NewProcess(command string, args []string, logger *log.Logger) (*Process, error) {
    cmd := exec.Command(command, args...)
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return nil, err
    }

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }

    stderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, err
    }

    return &Process{
        cmd:        cmd,
        Stdin:      stdin,
        Stdout:     stdout,
        Stderr:     stderr,
        StdinMutex: sync.Mutex{},
        DoneChan:   make(chan bool, 1),
        log:        logger,
    }, nil
}

func NewProcessMustSucceed(command string, args []string, logger *log.Logger) *Process {
    p, err := NewProcess(command, args, logger)
    if err != nil {
        panic(err)
    }
    return p
}

// Process is a representation of a command to be run and access to its stdin/out/err
type Process struct {
    cmd        *exec.Cmd
    Stderr     io.ReadCloser
    Stdout     io.ReadCloser
    Stdin      io.WriteCloser
    StdinMutex sync.Mutex
    DoneChan   chan bool
    log        *log.Logger
    hasStarted bool
}

func (p *Process) Start() error {
    p.log.Print("Starting")
    if err := p.cmd.Start(); err != nil {
        return fmt.Errorf("could not start process: %v", err)
    }
    return nil
}

func (p *Process) IsRunning() bool {
    return p.hasStarted && !p.cmd.ProcessState.Exited()
}

func (p *Process)GetProcStatus() string {
    return p.cmd.ProcessState.String()
}

// writes data to stdin on this process, adding a newline if one does not exist
func (p *Process) Write(data string) (int, error) {
    toWrite := data
    if !strings.HasSuffix(toWrite, "\n") {
        toWrite = toWrite + "\n"
    }

    p.StdinMutex.Lock()
    defer p.StdinMutex.Unlock()
    p.log.Printf("[STDIN] %s", data)
    return p.Stdin.Write([]byte(toWrite))
}

func (p *Process) WaitForCompletion() error {
    defer close(p.DoneChan)
    err := p.cmd.Wait()
    if err != nil {
        return err
    }
    return nil
}

// sends the given signal to the underlying process
func (p *Process) SendSignal(sig os.Signal) error {
    if !p.IsRunning() {
        p.log.Printf("[WARN] attempt to send non-running process a signal")
        return nil
    }

    if err := p.cmd.Process.Signal(sig); err != nil {
        p.log.Printf("[WARN] could not send signal %s to process: %s", sig.String(), err)
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
