package tomlconf

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// Config is the main config struct
type Config struct {
	OriginalPath string
	Connection   ConfigHolder

	FormatTemplates map[string]FormatSet         `toml:"format_templates"`
	RegexpTemplates map[string]map[string]Regexp `toml:"regexp_templates"`
	Games           map[string]*Game
}

func (c *Config) resolveImports() error {
	for gameName := range c.Games {
		game := c.Games[gameName] // because gocritic. and if I want to make changes I need a reference anyway
		if err := game.resolveImports(c); err != nil {
			return fmt.Errorf("unable to resolve imports for %q: %w", gameName, err)
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

	out, err := makeConfig(tree)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %w", err)
	}

	if err := out.resolveImports(); err != nil {
		return nil, fmt.Errorf("unable to resolve imports: %w", err)
	}

	if err := validateConfig(out); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return out, nil
}

func makeConfig(tree *toml.Tree) (*Config, error) {
	out := new(Config)
	if err := tree.Unmarshal(out); err != nil {
		return nil, err
	}

	if res, ok := tree.Get("connection.server").(*toml.Tree); ok {
		out.Connection.RealConf = res
	}

	for name, game := range out.Games {
		data := tree.GetPath([]string{"games", name, "transport"})
		if res, ok := data.(*toml.Tree); ok {
			game.Transport.RealConf = res
		}
	}

	return out, nil
}

func validateConfig(inConf *Config) error {
	if inConf.Connection.Type != "null" && inConf.Connection.RealConf == nil {
		return fmt.Errorf("invalid config for connection type %q, missing config", inConf.Connection.Type)
	}

	for n, g := range inConf.Games {
		if g.Transport.Type == "" || g.Transport.RealConf == nil {
			return fmt.Errorf("invalid config for game %q. Missing transport", n)
		}
	}
	return nil
}
