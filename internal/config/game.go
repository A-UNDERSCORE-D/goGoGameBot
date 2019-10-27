package config

import (
	"encoding/xml"
	"fmt"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format"
)

// GameManager represents a config for a game.Manager instance
type GameManager struct {
	XMLName    xml.Name `xml:"games"`
	StripMasks []string `xml:"strip_mask"` // TODO: move this to conn?
	Games      []Game   `xml:"game"`
}

// Game represents a config for a game.Game instance. It has embedded structs for organisation
type Game struct {
	XMLName xml.Name `xml:"game"`

	Name        string   `xml:"name,attr"`
	AutoRestart int      `xml:"auto_restart,attr"`
	AutoStart   bool     `xml:"auto_start,attr"`
	Path        string   `xml:"binary"`
	WorkingDir  string   `xml:"working_dir"`
	Args        string   `xml:"args"`
	Env         []string `xml:"environment"`
	DontCopyEnv bool     `xml:"dont_copy_env,attr"`
	PreRoll     struct {
		Regexp  string `xml:"regexp"`
		Replace string `xml:"replace"`
	} `xml:"pre_roll"`

	Commands []Command `xml:"command"`
	Regexps  []Regexp  `xml:"stdio_regexp"`

	ControlChannels struct {
		Admin string `xml:"admin"`
		Msg   string `xml:"msg"`
	} `xml:"status_channels"`

	Chat struct {
		DontBridge        bool     `xml:"dont_bridge,attr"`
		DontAllowForwards bool     `xml:"dont_allow_forwards,attr"`
		DumpStdout        bool     `xml:"dump_stdout,attr"`
		DumpStderr        bool     `xml:"dump_stderr,attr"`
		BridgedChannels   []string `xml:"bridged_channel"`
		Formats           struct {
			Message  *format.Format `xml:"message"`
			Join     *format.Format `xml:"join"`
			Part     *format.Format `xml:"part"`
			Nick     *format.Format `xml:"nick"`
			Quit     *format.Format `xml:"quit"`
			Kick     *format.Format `xml:"kick"`
			External *format.Format `xml:"external"`
			Extra    []*ExtraFormat `xml:"extra"`
		} `xml:"formats"`
		TransformerConfig TransformerConfig `xml:"transformer"`
	} `xml:"chat"`
}

// TransformerConfig holds configs for various implementations of Transformer.transformer
type TransformerConfig struct {
	Type   string `xml:"-"`
	Config string `xml:"-"`
}

// UnmarshalXML Implements the Unmarshaler interface in the XML library. Specifically
// this is designed to unmarshal a single attr while maintaining the content of the rest of the tag for
// later unmarshalinmg
func (t *TransformerConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if start.Name.Local != "transformer" {
		return nil
	}

	t.Type = "strip" // default to a strip transformer

	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			t.Type = attr.Value
			break
		}
	}

	res, err := reconstructXML(d, start)
	if err != nil {
		return fmt.Errorf("could not reconstruct XML for TransformerConfig: %w", err)
	}
	t.Config = res
	return nil
}

// ExtraFormat represents an additional format found in config.Game.Chat.Format
type ExtraFormat struct {
	format.Format
	Name string `xml:"name,attr"`
}

// Command represents a command that translates to input on the game's stdin
type Command struct {
	XMLName       xml.Name      `xml:"command"`
	Name          string        `xml:"name,attr"`
	RequiresAdmin int           `xml:"requires_admin,attr"`
	Help          string        `xml:"help"`
	Format        format.Format `xml:"format"`
}

// Regexp represents a format and a regex that are used together to forward specific lines formatted a specific way
// from the game's stdout to chat, other games, and itself
type Regexp struct {
	XMLName     xml.Name      `xml:"stdio_regexp"`
	Priority    int           `xml:"priority,attr"`
	Name        string        `xml:"name,attr"`
	Regexp      string        `xml:"regexp"`
	Format      format.Format `xml:"format"`
	DontEat     bool          `xml:"dont_eat,attr"`
	DontSend    bool          `xml:"dont_send_to_chan,attr"`
	DontForward bool          `xml:"dont_forward,attr"`
	SendToLocal bool          `xml:"send_to_local,attr"`
}
