package main

import (
    "bufio"
    "fmt"
    "github.com/A-UNDERSCORE-D/goGoGameBot/src/process"
    "github.com/chzyer/readline"
    "log"
    "sync"
    "time"
)

func init() {
    log.SetFlags(log.LstdFlags)

}

func main() {
    testProc, err := process.NewProcess("test", "/usr/bin/ncat", []string{"127.0.0.1", "1337"})
    if err != nil {
        panic(err)
    }

    rl, err := readline.New("> ")
    defer rl.Close()

    log.SetOutput(rl)
    wg := sync.WaitGroup{}
    wg.Add(1)
    go func() {
        for {
            line, err := rl.Readline()
            if err != nil {
                break
            }
            if line == "quit" {
                wg.Done()
                return
            }
            _, err = rl.Write([]byte(line))
            if err != nil {
                break
            }
        }
    }()

    go func() {
        s := bufio.NewScanner(testProc.Stdout)
        for s.Scan() {
            fmt.Println(s.Text())
        }
        fmt.Println("DONE")
    }()

    err = testProc.Start()
    if err != nil {
        panic(err)
    }

    _, err = testProc.Write("Here has a test message :D")
    if err != nil {
        panic(err)
    }

    go func() {
        for {
            select {
            case <- time.Tick(time.Second):
                log.Print(testProc.Status)
            }
        }
    }()
    wg.Wait()
}
