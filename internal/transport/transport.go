// Package transport provides an interface that allows for generic transport layers between gggb and a process
package transport

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"awesome-dragon.science/go/goGoGameBot/internal/config/tomlconf"
	"awesome-dragon.science/go/goGoGameBot/internal/interfaces"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/network"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/process"
	"awesome-dragon.science/go/goGoGameBot/internal/transport/util"
	"awesome-dragon.science/go/goGoGameBot/internal/version"
	"awesome-dragon.science/go/goGoGameBot/pkg/log"
)

// Transport is a way for a Game to talk to an underlying Process.
type Transport interface {
	// GetStatus returns the current state of the transport
	GetStatus() util.TransportStatus

	// GetHumanStatus returns the status of the transport in a human readable form
	GetHumanStatus() string

	// Stdout returns a channel that will have lines from stdout sent over it.
	// multiple calls are not supported. This channel should be closed once the source exits
	Stdout() <-chan []byte

	// Stderr returns a channel that will have lines from stdout sent over it.
	// multiple calls are not supported. This channel should be closed once the source exits
	Stderr() <-chan []byte

	// Update updates the Transport with a TransportConfig
	Update(tomlconf.ConfigHolder) error

	// StopOrKiller is for the Ability to stop the process on the other side of the Transport, as is Runner
	interfaces.StopOrKiller

	// Run runs the underlying process on the Transport. It returns the return code of the process (or -1 if start failed)
	// a string representation of the exit, if applicable, and an error. error should be checked first as the string
	// may not be filled for some errors.
	// The passed channel should be closed when the game is started, to allow the controller to start monitoring stdio.
	// the start chan MUST be closed sometime before run returns
	Run(start chan struct{}) (int, string, error)

	// IsRunning returns whether or not the underlying process is currently running. For more information use GetStatus.
	IsRunning() bool

	// Writer and StringWriter versions of all the write methods, for use with Fprintf etc
	io.Writer
	io.StringWriter
}

// ErrNoTransport indicates that a nonexistent transport was requested
var ErrNoTransport = errors.New("transport with that name does not exist")

// GetTransport returns a transport based on the given name.
func GetTransport(name string, transportConfig tomlconf.ConfigHolder, logger *log.Logger) (Transport, error) {
	switch strings.ToLower(name) {
	case "process":
		return process.New(transportConfig, logger)
	case "network":
		if !strings.HasPrefix(version.Version, "devel") {
			panic("Network transports are WIP, and not available in non-dev builds")
		}

		return network.New(transportConfig, logger)
	default:
		return nil, fmt.Errorf("cannot create transport %q: %w", name, ErrNoTransport)
	}
}
