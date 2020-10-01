package tomlconf

import (
	"fmt"
)

// Game holds the config for a Game instance
type Game struct {
	Name        string
	AutoStart   bool   `toml:"auto_start"`
	AutoRestart int    `toml:"auto_restart"`
	Comment     string `comment:"A message to be added to the status line of this Game"`

	Transport ConfigHolder

	PreRoll struct {
		Regexp  string
		Replace string
	} `comment:"regex to be applied to all outgoing lines before\n they are parsed by the below regexps"`

	Chat Chat `comment:"Configuration for how this game will interact with chat"`

	CommandImports []string           `toml:"import_commands"`
	Commands       map[string]Command `comment:"Commands that can be executed on the chat side"`

	RegexpImports []string `toml:"import_regexps"`
	Regexps       []Regexp `toml:"regexp"`
}

// Chat is a config for game.Chat
type Chat struct {
	BridgedChannel string `toml:"bridged_channel" comment:"The channel to bridge chat between"`
	// string ptr to check for null
	ImportFormat *string `toml:"import_format"`
	Formats      FormatSet

	BridgeChat    bool `toml:"bridge_chat" default:"true" comment:"Should this game bridge its chat (default true)"`
	DumpStdout    bool `toml:"dump_stdout" comment:"Dump stdout to the bridged channel (This is a spammy debug option)"`
	DumpStderr    bool `toml:"dump_stderr" comment:"Dump stdout to the bridged channel (This is a spammy debug option)"`
	AllowForwards bool `toml:"allow_forwards" default:"true" comment:"Allow messages from other games (default true)"`

	Transformer *ConfigHolder `comment:"How to transform messages to and from this game. (leave out for StripTransformer)"`
}

// Command holds commands that can be executed by users
type Command struct {
	Format        string `comment:"go template based formatter"`
	Help          string `comment:"help for the command"`
	RequiresAdmin int    `toml:"requires_admin" comment:"the admin level required to execute this command (0 for none)"`
}

// Regexp is a representation of a game regexp
type Regexp struct {
	Name   string
	Regexp string
	// TODO: maybe do a string pointer here, then checking in game will be easier when it comes to regexps
	// TODO: designed to eat things
	Format   string
	Priority int `toml:",omitempty"`

	Eat          bool `default:"true" comment:"Stop processing regexps after this is matched. (default true)"`
	SendToChan   bool `toml:"send_to_chan" default:"true" comment:"Send the formatted message to the bridged channel (default true)"`   //nolint:lll // Cant shorten them
	SendToOthers bool `toml:"send_to_others" default:"true" comment:"Send the formatted message to other running games (default true)"` //nolint:lll // Cant shorten them
	SendToLocal  bool `toml:"send_to_local" comment:"Send the formatted message to the game it came from (default false)"`
}

// FormatSet holds a set of formatters to be converted to a format.Format
type FormatSet struct {
	// string ptr to allow to check for null
	Message  *string
	Join     *string
	Part     *string
	Nick     *string
	Quit     *string
	Kick     *string
	External *string

	Extra map[string]string
}

// Indices into a FormatSet
const (
	MESSAGE = iota
	JOIN
	PART
	NICK
	QUIT
	KICK
	EXTERNAL
)

func (f *FormatSet) index(i int) *string {
	switch i {
	case MESSAGE:
		return f.Message
	case JOIN:
		return f.Join
	case PART:
		return f.Part
	case NICK:
		return f.Nick
	case QUIT:
		return f.Quit
	case KICK:
		return f.Kick
	case EXTERNAL:
		return f.External
	default:
		panic(fmt.Sprintf("Unexpected index %d into FormatSet", i))
	}
}

func (f *FormatSet) setIndex(i int, s *string) {
	switch i {
	case MESSAGE:
		f.Message = s
	case JOIN:
		f.Join = s
	case PART:
		f.Part = s
	case NICK:
		f.Nick = s
	case QUIT:
		f.Quit = s
	case KICK:
		f.Kick = s
	case EXTERNAL:
		f.External = s
	default:
		panic(fmt.Sprintf("Unexpected index %d into FormatSet", i))
	}
}

func (g *Game) resolveImports(c *Config) error {
	if err := g.resolveFormatImports(c); err != nil {
		return err
	}

	if err := g.resolveRegexpImports(c); err != nil {
		return err
	}

	if err := g.resolveCommandImports(c); err != nil {
		return err
	}

	return nil
}

func (g *Game) resolveFormatImports(c *Config) error {
	// TODO: store the current Chat.Formats to allow for overrides
	if g.Chat.ImportFormat == nil {
		return nil
	}

	currentFormats := g.Chat.Formats

	fmtTemplate, exists := c.FormatTemplates[*g.Chat.ImportFormat]
	if !exists {
		return fmt.Errorf(
			"could not resolve format import %q as it does not exist",
			*g.Chat.ImportFormat,
		)
	}

	g.Chat.Formats = fmtTemplate

	for i := MESSAGE; i <= EXTERNAL; i++ {
		if str := currentFormats.index(i); str != nil {
			g.Chat.Formats.setIndex(i, str)
		}
	}

	return nil
}

func (g *Game) resolveRegexpImports(c *Config) error {
	for _, templateName := range g.RegexpImports {
		importedRegexps, exists := c.RegexpTemplates[templateName]
		if !exists {
			return fmt.Errorf(
				"could not resolve regexp import %q as it does not exist",
				templateName,
			)
		}

		g.Regexps = append(g.Regexps, importedRegexps...)
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

		if g.Commands == nil {
			g.Commands = make(map[string]Command)
		}

		for name, command := range commands {
			g.Commands[name] = command
		}
	}

	return nil
}
