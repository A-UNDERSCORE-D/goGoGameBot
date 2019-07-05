package config

import (
	"encoding/xml"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

type GameManager struct {
	XMLName    xml.Name `xml:"games"`
	StripMasks []string `xml:"strip_masks>mask"`
	Games      []Game   `xml:"game"`
}

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
		StripMasks        []string `xml:"strip_mask"`
		BridgedChannels   []string `xml:"bridged_channel"`
		Formats           struct {
			Message  util.Format   `xml:"message"`
			Join     util.Format   `xml:"join"`
			Part     util.Format   `xml:"part"`
			Nick     util.Format   `xml:"nick"`
			Quit     util.Format   `xml:"quit"`
			Kick     util.Format   `xml:"kick"`
			External util.Format   `xml:"external"`
			Extra    []ExtraFormat `xml:"extra"`
		} `xml:"formats"`
	} `xml:"chat"`
	ColourMap ColourMap `xml:"colour_map"`
}

type ExtraFormat struct {
	util.Format
	Name string `xml:"name,attr"`
}

type Command struct {
	XMLName       xml.Name    `xml:"command"`
	Name          string      `xml:"name,attr"`
	RequiresAdmin int         `xml:"requires_admin,attr"`
	Help          string      `xml:"help"`
	Format        util.Format `xml:"format"`
}

type Regexp struct {
	XMLName     xml.Name    `xml:"stdio_regexp"`
	Priority    int         `xml:"priority,attr"`
	Name        string      `xml:"name,attr"`
	Regexp      string      `xml:"regexp"`
	Format      util.Format `xml:"format"`
	DontEat     bool        `xml:"dont_eat,attr"`
	DontSend    bool        `xml:"dont_send_to_chan,attr"`
	DontForward bool        `xml:"dont_forward,attr"`
	SendToLocal bool        `xml:"send_to_local,attr"`
}
