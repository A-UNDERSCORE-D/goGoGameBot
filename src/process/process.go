package process

import (
    "fmt"
    "github.com/chzyer/readline"
    "golang.org/x/sys/unix"
    "io"
    "log"
    "os"
    "os/exec"
    "strings"
    "sync"
    "time"
)

const (
    NotStarted = iota
    NotRunning
    Running
    Done
    Errored
)

func NewProcess(name string, command string, args []string, readline *readline.Instance) (*Process, error) {
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
        Name:       name,
        Cmd:        cmd,
        Stdin:      stdin,
        Stdout:     stdout,
        Stderr:     stderr,
        StdinMutex: sync.Mutex{},
        WriteChan:  make(chan string),
        Status:     NotStarted,
        DoneChan:   make(chan bool, 1),
        log:        log.New(readline, "["+name+"] ", log.Flags()),
    }, nil
}

func NewProcessMustSucceed(name, command string, args []string, readline *readline.Instance) *Process {
    p, err := NewProcess(name, command, args, readline)
    if err != nil {
        panic(err)
    }
    return p
}

// Process is a representation of a command to be run and access to its stdin/out/err
type Process struct {
    Name       string
    Cmd        *exec.Cmd
    Stderr     io.ReadCloser
    Stdout     io.ReadCloser
    Stdin      io.WriteCloser
    StdinMutex sync.Mutex
    WriteChan  chan string
    DoneChan   chan bool
    Status     int
    Err        error
    log        *log.Logger
}

func (p *Process) Start() error {
    p.log.Print("Starting")
    err := p.Cmd.Start()
    if err != nil {
        return fmt.Errorf("could not Start process %s: %v", p.Name, err)
    }
    go p.watchWriteChan()
    go p.waitOnProc()
    p.Status = Running
    p.log.Print("Started")
    return nil
}

func (p *Process) IsRunning() bool {
    return p.Status == Running
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

func (p *Process) waitOnProc() {
    err := p.Cmd.Wait()
    if err != nil {
        p.Status = Errored
        p.Err = err
        p.log.Printf("command returned an error: %v", err)
    } else {
        p.Status = Done
        p.Err = nil
        p.log.Print("command completed successfully")
    }
    p.DoneChan <- true
    close(p.WriteChan)
}

func (p *Process) watchWriteChan() {
loop:
    for {
        select {
        case data, ok := <-p.WriteChan:
            if !ok {
                p.log.Print("WriteChan on Process closed")
                break loop
            }

            _, err := p.Write(data)
            if err != nil {
                p.log.Printf("Could not Write %q to stdin: %v", data, err)
            }
        }
    }
}

// sends the given signal to the underlying process
func (p *Process) SendSignal(sig os.Signal) error {
    if !p.IsRunning() {
        p.log.Printf("[WARN] attempt to send non-running process a signal")
        return nil
    }

    if err := p.Cmd.Process.Signal(sig); err != nil {
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
