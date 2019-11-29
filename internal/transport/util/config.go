package util

import "errors"

// Various errors for help with signaling erroneous state
var (
	ErrorAlreadyRunning = errors.New("already running")
	ErrorNotRunning     = errors.New("not running")
)

type BaseConfig struct {
	Path             string   `xml:"binary"`
	Args             string   `xml:"args"`
	WorkingDirectory string   `xml:"working_directory"`
	Environment      []string `xml:"environment"`
	DontCopyEnv      bool     `xml:"should_copy_environment"`
}
