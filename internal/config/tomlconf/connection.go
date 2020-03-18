package tomlconf

import "github.com/pelletier/go-toml"

// Connection holds the config for the interfaces.Bot implementation in use by
// GGGB
type Connection struct {
	Type string `toml:"type" default:"null" comment:"The type of connection to use"`
	// This apparently doesn't work but I have an open issue about it.
	// until that issue is resolved, I have done the resolution here manually in
	// GetRegexp
	RawConf *toml.Tree `toml:"server"`
}
