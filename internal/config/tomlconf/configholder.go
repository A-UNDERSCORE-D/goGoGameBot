package tomlconf

import "github.com/pelletier/go-toml"

type ConfigHolder struct {
	Type     string `toml:"type"`
	RealConf *toml.Tree
}
