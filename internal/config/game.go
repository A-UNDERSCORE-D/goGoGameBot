package config

import (
	"encoding/xml"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util/format"
)

// GameManager represents a config for a game.Manager instance
type GameManager struct {
	XMLName    xml.Name `xml:"games"`
	StripMasks []string `xml:"strip_mask"` // TODO: move this to conn?
	Games      []Game   `xml:"game"`
}

// Game represents a config for a game.Game instance. It has embedded structs for organisation
type Game struct {
	XMLName         xml.Name  `xml:"game"`
	Name            string    `xml:"name,attr"`
	AutoRestart     int       `xml:"auto_restart,attr"`
	AutoStart       bool      `xml:"auto_start,attr"`
	Path            string    `xml:"binary"`
	WorkingDir      string    `xml:"working_dir"`
	Args            string    `xml:"args"`
	Commands        []Command `xml:"command"`
	Regexps         []Regexp  `xml:"stdio_regexp"`
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
			Message  format.Format `xml:"message"`
			Join     format.Format `xml:"join"`
			Part     format.Format `xml:"part"`
			Nick     format.Format `xml:"nick"`
			Quit     format.Format `xml:"quit"`
			Kick     format.Format `xml:"kick"`
			External format.Format `xml:"external"`
			Extra    []ExtraFormat `xml:"extra"`
		} `xml:"formats"`
	} `xml:"chat"`
	ColourMap ColourMap `xml:"colour_map"`
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
