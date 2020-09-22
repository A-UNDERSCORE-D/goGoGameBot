package tomlconf

import "github.com/pelletier/go-toml"

// ConfigHolder holds a config that is unknown at parse time
type ConfigHolder struct {
	Type     string     `toml:"type"`
	RealConf *toml.Tree `toml:"config"`
}
