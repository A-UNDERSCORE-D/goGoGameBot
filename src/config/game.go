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

func (g *GameRegexp) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
    type gameRegexp GameRegexp                       // Dont cause recursion when we use decode element later
    out := gameRegexp{Priority: -1, ShouldEat: true} // Set some default values that are different from the zero value of their types
    if err := d.DecodeElement(&out, &start); err != nil {
        return err
    }

    *g = (GameRegexp)(out)
    return nil
}

type Game struct {
    XMLName      xml.Name     `xml:"game"`
    Name         string       `xml:"name,attr"`
    AutoStart    bool         `xml:"auto_start,attr"`
    Path         string       `xml:"bin_path,attr"`
    Args         string       `xml:"args,attr"`
    Logchan      string       `xml:"log_chan,attr"`
    AdminLogChan string       `xml:"admin_log_chan,attr"`
    LogStdout    bool         `xml:"log_stdout,attr"`
    LogStderr    bool         `xml:"log_stderr,attr"`
    Regexps      []GameRegexp `xml:"regexp"`
}
