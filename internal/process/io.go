package process

import (
	"fmt"
	"io"
	"sync"
)

func waitGroupIoCopy(wg *sync.WaitGroup, src io.Reader) io.Reader {
	pipeR, pipeW := io.Pipe()

	go func() {
		if _, err := io.Copy(pipeW, src); err != nil {
			fmt.Printf("Warning: Error from io.Copy: %s", err)
		}

		pipeW.Close()

		wg.Done()
	}()

	return pipeR
}
