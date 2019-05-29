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
	XMLName        xml.Name  `xml:"game"`
	Name           string    `xml:"name,attr"`
	Parent         string    `xml:"parent"`
	AutoRestart    int       `xml:"auto_restart,attr"`
	AutoStart      bool      `xml:"auto_start,attr"`
	Commands       []Command `xml:"commands>command"`
	Regexps        []Regexp
	StatusChannels struct {
		Admin string `xml:"admin"`
		Msg   string `xml:"msg"`
	} `xml:"status_channels"`

	Chat struct {
		DontBridge        bool     `xml:"dont_bridge,attr"`
		DontAllowForwards bool     `xml:"dont_allow_forwards,attr"`
		DumpStdout        bool     `xml:"dump_stdout,attr"`
		DumpStderr        bool     `xml:"dump_stderr,attr"`
		StripMasks        []string `xml:"strip_masks>mask"`
		BridgedChannels   []string `xml:"bridged_channels>channel"`
		Formats           struct {
			Normal   util.Format `xml:"normal"`
			JoinPart util.Format `xml:"join_part"`
			Nick     util.Format `xml:"nick"`
			Quit     util.Format `xml:"quit"`
			Kick     util.Format `xml:"kick"`
			External util.Format `xml:"external"`
		} `xml:"formats"`
	}
	ColourMap ColourMap `xml:"colour_map"`
}

type Command struct {
	XMLName       xml.Name    `xml:"command"`
	Name          string      `xml:"name"`
	RequiresAdmin int         `xml:"requires_admin"`
	Help          string      `xml:"help"`
	Format        util.Format `xml:"format"`
}

type Regexp struct {
	XMLName     xml.Name    `xml:"regexp"`
	Priority    int         `xml:"priority"`
	Name        string      `xml:"name"`
	Regexp      string      `xml:"regexp"`
	Format      util.Format `xml:"format"`
	DontEat     bool        `xml:"dont_eat"`
	DontSend    bool        `xml:"dont_send_to_chan"`
	DontForward bool        `xml:"dont_forward"`
}
