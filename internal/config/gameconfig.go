package config

import (
	"encoding/xml"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

type GameRegexpConfig struct {
	XMLName         xml.Name    `xml:"game_regexp"`
	Name            string      `xml:"name,attr"`
	Priority        int         `xml:"priority,attr"`
	ShouldEat       bool        `xml:"should_eat,attr"`
	Regexp          string      `xml:"regexp"`
	Format          util.Format `xml:"format"`
	SendToChan      bool        `xml:"send_to_chan,attr"`
	ForwardToOthers bool        `xml:"forward_to_others,attr"`
}

func (g *GameRegexpConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type gameRegexp GameRegexpConfig // Dont cause recursion when we use decode element later
	// Set some default values that are different from the zero value of their types
	out := gameRegexp{Priority: -1, ShouldEat: true, SendToChan: true}

	if err := d.DecodeElement(&out, &start); err != nil {
		return err
	}

	*g = (GameRegexpConfig)(out)
	return nil
}

type GameCommandConfig struct {
	Name          string      `xml:"name,attr"`
	StdinFormat   util.Format `xml:"format"`
	RequiresAdmin bool        `xml:"requires_admin,attr"`
}

type GameConfig struct {
	XMLName            xml.Name           `xml:"game"`
	Include            string             `xml:"include,attr,omitempty"`
	IncludeRegexp      string             `xml:"include_regexp,attr,omitempty"`
	Name               string             `xml:"name,attr"`
	AutoStart          bool               `xml:"auto_start,attr"`
	RestartOnCleanExit bool               `xml:"restart_on_clean_exit,attr"`
	Path               string             `xml:"bin_path,attr"`
	WorkingDir         string             `xml:"working_dir,attr"`
	Args               string             `xml:"args,attr"`
	LogChan            string             `xml:"log_chan,attr"`
	AdminLogChan       string             `xml:"admin_log_chan,attr"`
	LogStdout          bool               `xml:"log_stdout,attr"`
	LogStderr          bool               `xml:"log_stderr,attr"`
	BridgeChat         bool               `xml:"bridge_chat,attr"`
	Regexps            []GameRegexpConfig `xml:"game_regexp"`
	BridgeChans        []string           `xml:"bridge_chan"`
	BridgeFmt          util.Format        `xml:"bridge_format"`
	JoinPartFmt        util.Format        `xml:"join_part_format"`
	OtherForwardFmt    util.Format        `xml:"other_forward_format"`

	ColourMap ColourMap           `xml:"colour_map,omitempty"`
	Commands  []GameCommandConfig `xml:"command"`
}

func (g *GameConfig) doInclude() (*GameConfig, error) {
	if err := g.includeFromFile(); err != nil {
		return nil, err
	}
	if err := g.doIncludeRegexps(); err != nil {
		return nil, err
	}
	return g, nil
}

// TODO: It would be nice if this wouldn't overwrite anything that has been set
//       But doing that isn't exactly simple. For now it just overwrites stuff set
//       in the included file, ie, if you set something and its set in the included
//       file, the stuff in the included file wins
func (g *GameConfig) includeFromFile() error {
	if g.Include == "" {
		return nil
	}

	data, err := readAllFromFile(g.Include)
	if err != nil {
		return err
	}

	toSet := *g

	if err := xml.Unmarshal(data, &toSet); err != nil {
		return err
	}
	*g = toSet
	return nil
}

type toInclude struct {
	XMLName xml.Name           `xml:"regexps"`
	Regexps []GameRegexpConfig `xml:"game_regexp"`
}

func (g *GameConfig) doIncludeRegexps() error {
	if g.IncludeRegexp == "" {
		return nil
	}

	data, err := readAllFromFile(g.IncludeRegexp)
	if err != nil {
		return err
	}

	var toAdd toInclude
	//var toAdd []GameRegexpConfig
	if err := xml.Unmarshal(data, &toAdd); err != nil {
		return err
	}

	g.Regexps = append(g.Regexps, toAdd.Regexps...)
	return nil
}

type ColourMap struct {
	Bold          string `xml:"bold,omitempty"`
	Italic        string `xml:"italic,omitempty"`
	ReverseColour string `xml:"reverse_colour,omitempty"`
	Strikethrough string `xml:"strikethrough,omitempty"`
	Underline     string `xml:"underline,omitempty"`
	Monospace     string `xml:"monospace,omitempty"`
	Reset         string `xml:"reset,omitempty"`
	White         string `xml:"white,omitempty"`
	Black         string `xml:"black,omitempty"`
	Blue          string `xml:"blue,omitempty"`
	Green         string `xml:"green,omitempty"`
	Red           string `xml:"red,omitempty"`
	Brown         string `xml:"brown,omitempty"`
	Magenta       string `xml:"magenta,omitempty"`
	Orange        string `xml:"orange,omitempty"`
	Yellow        string `xml:"yellow,omitempty"`
	LightGreen    string `xml:"light_green,omitempty"`
	Cyan          string `xml:"cyan,omitempty"`
	LightCyan     string `xml:"light_cyan,omitempty"`
	LightBlue     string `xml:"light_blue,omitempty"`
	Pink          string `xml:"pink,omitempty"`
	Grey          string `xml:"grey,omitempty"`
	LightGrey     string `xml:"light_grey,omitempty"`
}

func (c *ColourMap) ToMap() map[string]string {
	return map[string]string{
		"$$":              "$",
		"$b":              c.Bold,
		"$i":              c.Italic,
		"$v":              c.ReverseColour,
		"$s":              c.Strikethrough,
		"$u":              c.Underline,
		"$m":              c.Monospace,
		"$r":              c.Reset,
		"$c[white]":       c.White,
		"$c[black]":       c.Black,
		"$c[blue]":        c.Blue,
		"$c[green]":       c.Green,
		"$c[red]":         c.Red,
		"$c[brown]":       c.Brown,
		"$c[magenta]":     c.Magenta,
		"$c[orange]":      c.Orange,
		"$c[yellow]":      c.Yellow,
		"$c[light green]": c.LightGreen,
		"$c[cyan]":        c.Cyan,
		"$c[light cyan]":  c.LightCyan,
		"$c[light blue]":  c.LightBlue,
		"$c[pink]":        c.Pink,
		"$c[grey]":        c.Grey,
		"$c[light grey]":  c.LightGrey,
	}
}
