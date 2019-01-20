package process

import (
    "errors"
    "fmt"
    "git.ferricyanide.solutions/A_D/goGoGameBot/src/util/botLog"
    "golang.org/x/sys/unix"
    "io"
    "os"
    "os/exec"
    "strings"
    "sync"
    "time"
)

func NewProcess(command string, args []string, logger *botLog.Logger) (*Process, error) {

    p := &Process{
        commandString: command,
        argListString: args,
        StdinMutex:    sync.Mutex{},
        log:           logger,
    }
    if err := p.Reset(); err != nil{
        return nil, err
    }
    return p, nil
}

func NewProcessMustSucceed(command string, args []string, logger *botLog.Logger) *Process {
    p, err := NewProcess(command, args, logger)
    if err != nil {
        panic(err)
    }
    return p
}

// Process is a representation of a command to be run and access to its stdin/out/err
type Process struct {
    cmd           *exec.Cmd
    commandString string
    argListString []string
    commandMutex sync.Mutex
    Stderr       io.ReadCloser
    Stdout       io.ReadCloser
    Stdin        io.WriteCloser
    StdinMutex   sync.Mutex
    DoneChan     chan bool
    log          *botLog.Logger
    hasStarted   bool
    hasExited    bool
}

// UpdateCmd sets the command and arguments to be used when creating the exec.Cmd used internally.
// It is safe for concurrent use
func (p *Process) UpdateCmd(command string, args []string) {
    p.commandMutex.Lock()
    defer p.commandMutex.Unlock()
    p.commandString = command
    p.argListString = args
}

func (p *Process) setupCmd() error {
    p.commandMutex.Lock()
    cmd := exec.Command(p.commandString, p.argListString...)
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
    p.DoneChan = make(chan bool)
    p.hasStarted = false
    p.hasExited = false
    return p.setupCmd()
}

func (p *Process) Start() error {
    p.log.Info("Starting")
    if err := p.cmd.Start(); err != nil {
        return fmt.Errorf("could not start process: %v", err)
    }
    p.hasStarted = true
    return nil
}

func (p *Process) IsRunning() bool {
    return p.hasStarted && !p.hasExited
}

func (p *Process) GetProcStatus() string {
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
    p.log.Infof("[STDIN] %s", data)
    return p.Stdin.Write([]byte(toWrite))
}

func (p *Process) WaitForCompletion() error {
    defer close(p.DoneChan)
    err := p.cmd.Wait()
    p.hasExited = true
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
