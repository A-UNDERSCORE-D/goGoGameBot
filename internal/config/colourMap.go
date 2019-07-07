package config

// ColourMap is map to convert IRC colours to other formats
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

// ToMap converts the ColourMap struct to a map of string to string. It additionally adds a mapping of "$$" -> "$"
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
