package config

import "encoding/xml"

type GameRegexp struct {
    XMLName   xml.Name `xml:"game_regexp"`
    Name      string   `xml:"name,attr"`
    Priority  int      `xml:"priority,attr"`
    ShouldEat bool     `xml:"should_eat,attr"`
    Regexp    string   `xml:"regexp"`
    Format    string   `xml:"format"`
}

type Game struct {
    XMLName      xml.Name     `xml:"game"`
    Name         string       `xml:"name,attr"`
    AutoStart    bool         `xml:"auto_start,attr"`
    Path         string       `xml:"bin_path,attr"`
    Args         string       `xml:"args,attr"`
    Logchan      string       `xml:"log_chan,attr"`
    AdminLogChan string       `xml:"admin_log_chan,attr"`
    Regexps      []GameRegexp `xml:"regexp"`
}
