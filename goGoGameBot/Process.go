package main

import (
    "io"
    "log"
    "os/exec"
    "strings"
    "sync"
)

type Process struct {
    Name        string
    cmd         *exec.Cmd
    Stderr      io.Reader
    StderrMutex sync.Mutex
    Stdout      io.Reader
    StdoutMutex sync.Mutex
    Stdin       io.Writer
    StdinMutex  sync.Mutex
    WriteChan   chan string
}

func (p *Process) write(data string) (int, error) {
    toWrite := data
    if !strings.HasSuffix(toWrite, "\n") {
        toWrite = toWrite + "\n"
    }

    p.StdinMutex.Lock()
    defer p.StdinMutex.Unlock()
    log.Printf("[%s] [STDIN] %s", p.Name, data)
    return p.Stdin.Write([]byte(toWrite))
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
            _, err := p.write(data)
            if err != nil {
                log.Printf("[%s] Could not write %q to stdin: %v", p.Name, data, err)
            }
        }
    }
}
