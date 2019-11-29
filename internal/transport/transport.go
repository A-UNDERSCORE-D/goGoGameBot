package transport

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/interfaces"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/processTransport"
	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/transport/util"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/log"
)

// Transport is a way for a Game to talk to an underlying Process.
type Transport interface {
	// GetStatus returns the current state of
	GetStatus() util.TransportStatus

	// GetHumanStatus returns the status of the transport that is human readable
	GetHumanStatus() string

	// Stdout returns a channel that will have lines from stdout sent over it.
	// It is valid to call Stdout multiple times for one instance, the lines will be
	// fanned out over every channel. Close the channel if you want to stop getting lines
	Stdout() <-chan []byte

	// Stderr returns a channel that will have lines from stdout sent over it.
	// It is valid to call Stdout multiple times for one instance, the lines will be
	// fanned out over every channel. Close the channel if you want to stop getting lines
	Stderr() <-chan []byte

	// Update updates the Transport with a TransportConfig
	Update(config.TransportConfig) error

	// StopOrKiller is for the Ability to stop the process on the other side of the Transport, as is Runner
	interfaces.StopOrKiller

	// Run runs the underlying process on the Transport. It returns the return code of the process (or -1 if start failed)
	// a string representation of the exit, if applicable, and an error. error should be checked first as the string
	// may not be filled for some errors.
	Run() (int, string, error)

	// IsRunning returns whether or not the underlying process is currently running. For more information use GetStatus.
	IsRunning() bool

	// Writer and StringWriter versions of all the write methods, for use with Fprintf etc
	io.Writer
	io.StringWriter
}

var ErrNoTransport = errors.New("transport with that name does not exist")

// GetTransport returns a transport based on the given name.
func GetTransport(name string, transportConfig config.TransportConfig, logger *log.Logger) (Transport, error) {
	switch strings.ToLower(name) {
	case "process":
		return processTransport.New(transportConfig, logger)
	default:
		return nil, fmt.Errorf("cannot create transport %q: %w", name, ErrNoTransport)
	}
}
