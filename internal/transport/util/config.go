package util

import "errors"

// Various errors for help with signaling erroneous state
var (
	ErrorAlreadyRunning = errors.New("already running")
	ErrorNotRunning     = errors.New("not running")
)

type BaseConfig struct {
	Path             string   `toml:"path" comment:"path to binary"`
	Args             string   `toml:"args" comment:"args to binary"`
	WorkingDirectory string   `toml:"working_directory" comment:"working directory for binary"`
	Environment      []string `toml:"environment" comment:"environment variables to add to the execution"`
	CopyEnv          bool     `toml:"copy_env" comment:"copy the environment of the bot when creating "`
}
