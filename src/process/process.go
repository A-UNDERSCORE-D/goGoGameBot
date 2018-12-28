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

const (
    NotStarted = iota
    NotRunning
    Running
    Done
    Errored
)

func NewProcess(name string, command string, args []string) (*Process, error) {
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
    }, nil
}

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
}

func (p *Process) Start() error {
    err := p.Cmd.Start()
    if err != nil {
        return fmt.Errorf("could not Start process %s: %v", p.Name, err)
    }
    go p.watchWriteChan()
    go p.waitOnProc()
    p.Status = Running
    return nil
}

func (p *Process) log(msg string) {
    log.Printf("[%s] %s", p.Name, msg)
}

func (p *Process) logf(formatStr string, args ...interface{}) {
    p.log(fmt.Sprintf(formatStr, args...))
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
    p.logf("[STDIN] %s", data)
    return p.Stdin.Write([]byte(toWrite))
}

func (p *Process) waitOnProc() {
    err := p.Cmd.Wait()
    if err != nil {
        p.Status = Errored
        p.Err = err
        p.logf("command returned an error: %v", err)
    } else {
        p.Status = Done
        p.Err = nil
        p.logf("command completed successfully")
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
                log.Printf("[%s] WriteChan on Process closed", p.Name)
                break loop
            }

            _, err := p.Write(data)
            if err != nil {
                log.Printf("[%s] Could not Write %q to stdin: %v", p.Name, data, err)
            }
        }
    }
}

// sends the given signal to the underlying process
func (p *Process) SendSignal(sig os.Signal) error {
    if !p.IsRunning() {
        p.log("[WARN] attempt to send non-running process a signal")
        return nil
    }

    if err := p.Cmd.Process.Signal(sig); err != nil {
        p.logf("[WARN] could not send signal %s to process: %s", sig.String(), err)
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
