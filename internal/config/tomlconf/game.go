package tomlconf

import (
	"fmt"
)

// Game holds the config for a Game instance
type Game struct {
	AutoStart   bool
	AutoRestart int

	Transport ConfigHolder

	StatusChannels struct {
		Admin string
		Msg   string
	} `toml:"status_channels"`

	Chat struct {
		BridgedChannel string `toml:"bridged_channel"`
		// string ptr to check for null
		ImportFormat *string `toml:"import_format"`
		Formats      FormatSet
	}

	Commands map[string]struct {
		Format        string
		Help          string
		RequiresAdmin int `toml:"requires_admin"`
	}

	RegexpImports []string          `toml:"import_regexps"`
	Regexps       map[string]Regexp `toml:"regexp"`
}

// Regexp is a representation of a game regexp
type Regexp struct {
	Regexp   string `toml:"regexp"`
	Format   string
	Priority int `toml:",omitempty"`
}

type FormatSet struct {
	Message string
	Join    string
	Part    string
	Nick    string
	Quit    string
	Kick    string
	Extra   map[string]string
}

func (g *Game) resolveImports(c *Config) error {
	if g.Chat.ImportFormat == nil {
		return nil
	}

	fmtTemplate, exists := c.FormatTemplates[*g.Chat.ImportFormat]
	if !exists {
		return fmt.Errorf(
			"could not resolve format import %q as it does not exist",
			*g.Chat.ImportFormat,
		)
	}
	g.Chat.Formats = fmtTemplate

	for _, templateName := range g.RegexpImports {
		regexpSet, exists := c.RegexpTemplates[templateName]
		if !exists {
			return fmt.Errorf(
				"could not resolve regexp import %q as it does not exist",
				templateName,
			)
		}

		for name, regexp := range regexpSet {
			g.Regexps[name] = regexp
		}
	}
	return nil
}
