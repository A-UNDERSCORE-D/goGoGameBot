package tomlconf

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// Config is the main config struct
type Config struct {
	OriginalPath string `toml:"-"`
	Connection   ConfigHolder

	FormatTemplates  map[string]FormatSet          `toml:"format_templates"`
	RegexpTemplates  map[string][]Regexp           `toml:"regexp_templates"`
	CommandTemplates map[string]map[string]Command `toml:"command_templates"`
	Games            []*Game                       `toml:"game"`
}

func (c *Config) resolveImports() error {
	for idx := range c.Games {
		game := c.Games[idx] // because gocritic. and if I want to make changes I need a reference anyway
		if err := game.resolveImports(c); err != nil {
			return fmt.Errorf("unable to resolve imports for %q: %w", game.Name, err)
		}
	}

	return nil
}

// GetConfig fetches the config located at the given path
func GetConfig(path string) (*Config, error) {
	tree, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read or parse config file: %w", err)
	}

	out, err := configFromTree(tree)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}

	if err := out.resolveImports(); err != nil {
		return nil, fmt.Errorf("unable to resolve imports: %w", err)
	}

	if err := validateConfig(out); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	out.OriginalPath = path

	return out, nil
}

func configFromTree(tree *toml.Tree) (*Config, error) {
	out := new(Config)
	if err := tree.Unmarshal(out); err != nil {
		return nil, err
	}

	return out, nil
}

func validateConfig(inConf *Config) error {
	if inConf.Connection.Type != "null" && inConf.Connection.RealConf == nil {
		return fmt.Errorf("invalid config for connection type %q, missing config", inConf.Connection.Type)
	}

	for _, g := range inConf.Games {
		if g.Transport.Type == "" || g.Transport.RealConf == nil {
			return fmt.Errorf("invalid config for game %q. Missing transport", g.Name)
		}
	}

	return nil
}
