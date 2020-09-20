package game

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
)

func (g *Game) watchStdinChan() {
	for toSend := range g.stdinChan {
		toSend = append(bytes.Trim(toSend, "\r\n"), '\n')
		if _, err := g.transport.Write(toSend); err != nil {
			g.manager.Error(fmt.Errorf("could not write to stdin chan for %q: %s", g.name, err))
		}
	}
}

// Write writes the given data to the transport's STDIN, it it safe to use concurrently
func (g *Game) Write(p []byte) (n int, err error) {
	if !g.IsRunning() {
		return 0, errors.New("cannot write to a non-running game")
	}
	g.stdinChan <- p

	return len(p), nil
}

// WriteString is the same as Write but accepts a string instead of a byte slice
func (g *Game) WriteString(s string) (n int, err error) {
	return g.Write([]byte(s))
}

func (g *Game) monitorStdIO(start chan struct{}, wg *sync.WaitGroup) {
	<-start

	if !g.IsRunning() {
		g.manager.Error(errors.New(g.prefixMsg("cannot watch stdio on a non-running game")))
		return
	}

	monitorChan := func(c <-chan []byte, stdout bool, wg *sync.WaitGroup) {
		defer wg.Done()

		for b := range c {
			g.handleStdIO(string(b), stdout)
		}
	}

	go monitorChan(g.transport.Stdout(), true, wg)
	go monitorChan(g.transport.Stderr(), false, wg)
}

const (
	stdout = "[STDOUT]"
	stderr = "[STDERR]"
)

func pickString(s1, s2 string, b bool) string {
	if b {
		return s1
	}

	return s2
}

func (g *Game) handleStdIO(text string, isStdout bool) {
	if g.preRollRe != nil {
		g.Tracef("prePreRoll: %s", text)
		text = g.preRollRe.ReplaceAllString(text, g.preRollReplace)
	}

	text = g.chatBridge.transformer.MakeIntermediate(text)

	g.Info(pickString(stdout, stderr, isStdout), " ", text)

	if (g.chatBridge.dumpStdout && isStdout) || (g.chatBridge.dumpStderr && !isStdout) {
		g.sendToMsgChan(pickString(stdout, stderr, isStdout), " ", text)
	}

	g.regexpManager.checkAndExecute(text, isStdout)
}
