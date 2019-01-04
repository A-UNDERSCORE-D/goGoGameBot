package process

import (
    "bufio"
    "fmt"
    "log"
    "sync"
    "time"
)

func NewManager(logger *log.Logger, processes ...*Process) *Manager {
    return &Manager{Processes: processes, log: logger}
}

type Manager struct {
    Processes []*Process
    log       *log.Logger
}

func (m *Manager) FanOutWrite(data string) {
    for _, p := range m.Processes {
        p.WriteChan <- data
    }
}

func (m *Manager) StartAllProcesses() {
    m.StartAllProcessesDelay(0)
}

func (m *Manager) StartAllProcessesDelay(delay time.Duration) {
    for _, p := range m.Processes {
        err := p.Start()
        m.log.Printf("Starting process %q", p.Name)
        if err != nil {
            m.log.Printf("[WARN] Could not start process %q: %s", p.Name, err)
            continue
        }
        go m.watchProcStdout(p)
        time.Sleep(delay)
    }
}

func (m *Manager) StopOrKillAllProcesses() {
    wg := sync.WaitGroup{}
    for _, p := range m.Processes {
        wg.Add(1)
        go func() {
            _ = p.StopOrKillTimeout(time.Second * 60)
            wg.Done()
        }()
    }
    wg.Wait()
}

func (m *Manager) hasProcess(proc *Process) bool {
    for _, p := range m.Processes {
        if p == proc {
            return true
        }
    }
    return false
}

func (m *Manager) GetProcessByName(name string) (*Process, error) {
    for _, p := range m.Processes {
        if p.Name == name {
            return p, nil
        }
    }
    return nil, fmt.Errorf("could not find process with name %q", name)
}

func (m *Manager) WriteToProcess(name, data string) error {
    p, err := m.GetProcessByName(name)
    if err != nil {
        return err
    }

    p.WriteChan <- data

    return nil
}

func (m *Manager) watchProcStdout(proc *Process) {
    if !m.hasProcess(proc) {
        m.log.Printf("[WARN] attempt to watch stdout on a process we dont control")
    }
    scanner := bufio.NewScanner(proc.Stdout)
    for scanner.Scan() {
        data := scanner.Text()
        proc.log.Printf("[stdout] %s", data)
    }
}
