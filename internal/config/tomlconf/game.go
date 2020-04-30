package tomlconf

import (
	"fmt"
)

// Game holds the config for a Game instance
type Game struct {
	AutoStart   bool
	AutoRestart int

	Transport ConfigHolder

	PreRoll struct {
		Regexp  string
		Replace string
	}

	Chat Chat

	CommandImports []string
	Commands       map[string]Command

	RegexpImports []string          `toml:"import_regexps"`
	Regexps       map[string]Regexp `toml:"regexp"`
}

type Chat struct {
	BridgedChannel string `toml:"bridged_channel"`
	AdminChannel   string `toml:"admin_channel"`
	// string ptr to check for null
	ImportFormat *string `toml:"import_format"`
	Formats      FormatSet
}

type Command struct {
	Format        string `comment:"go template based formatter"`
	Help          string `comment:"help for the command"`
	RequiresAdmin int    `toml:"requires_admin" comment:"the admin level required to execute this command (0 for none)"`
}

// Regexp is a representation of a game regexp
type Regexp struct {
	Regexp   string `toml:"regexp"`
	Format   string
	Priority int `toml:",omitempty"`
}

// FormatSet holds a set of formatters to be converted to a format.Format
type FormatSet struct { // TODO: use a Format from format? should be easy to do
	Message string
	Join    string
	Part    string
	Nick    string
	Quit    string
	Kick    string
	Extra   map[string]string
}

// TODO: Most of the dont* values here were done to avoid missing defaults.
// TODO: they could probably be changed to use a default as provided by the toml lib
type gameChat struct {
	DontBridge        bool   `toml:"dont_bridge" comment:"should chat be bridged from the chat platform?"`
	DontAllowForwards bool   `toml:"dont_allow_forwards" comment:"should chat be forwarded from other games"`
	DumpStdout        bool   `toml:"dump_stdout" comment:"should stdout be dumped to the chat platform?"`
	DumpStderr        bool   `toml:"dump_stderr" comment:"should stderr be dumped to the chat platform?"`
	BridgedChannel    string `toml:"bridged_channel" comment:"The chat channel to bridge game chat to"`
	Formats           FormatSet
	ImportFormat      *string `toml:"import_format" comment:"Template to import formats from"`
}

func (g *Game) resolveImports(c *Config) error {
	if err := g.resolveFormatImports(c); err != nil {
		return err
	}

	if err := g.resolveFormatImports(c); err != nil {
		return err
	}

	if err := g.resolveCommandImports(c); err != nil {
		return err
	}

	return nil
}

func (g *Game) resolveFormatImports(c *Config) error {
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
	return nil
}

func (g *Game) resolveRegexpImports(c *Config) error {
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

func (g *Game) resolveCommandImports(c *Config) error {
	for _, templateName := range g.CommandImports {
		commands, exists := c.CommandTemplates[templateName]
		if !exists {
			return fmt.Errorf(
				"could not resolve command import %q as it does not exist",
				templateName,
			)
		}

		for name, command := range commands {
			g.Commands[name] = command
		}
	}
	return nil
}
