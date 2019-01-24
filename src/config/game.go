package config

import (
    "encoding/xml"
)

type GameRegexp struct {
    XMLName    xml.Name `xml:"game_regexp"`
    Name       string   `xml:"name,attr"`
    Priority   int      `xml:"priority,attr"`
    ShouldEat  bool     `xml:"should_eat,attr"`
    Regexp     string   `xml:"regexp"`
    Format     string   `xml:"format"`
    SendToChan bool     `xml:"send_to_chan,attr"`
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
    XMLName       xml.Name     `xml:"game"`
    Include       string       `xml:"include,attr,omitempty"`
    IncludeRegexp string       `xml:"include_regexp,attr,omitempty"`
    Name          string       `xml:"name,attr"`
    AutoStart     bool         `xml:"auto_start,attr"`
    Path          string       `xml:"bin_path,attr"`
    WorkingDir    string       `xml:"working_dir,attr"`
    Args          string       `xml:"args,attr"`
    Logchan       string       `xml:"log_chan,attr"`
    AdminLogChan  string       `xml:"admin_log_chan,attr"`
    LogStdout     bool         `xml:"log_stdout,attr"`
    LogStderr     bool         `xml:"log_stderr,attr"`
    Regexps       []GameRegexp `xml:"game_regexp"`
    BridgeChat    bool         `xml:"bridge_chat,attr"`
    BridgeChans   []string     `xml:"bridge_chan"`
    BridgeFmt     string       `xml:"bridge_format"`
}

func (g *Game) doInclude() (*Game, error) {
    if err := g.includeFromFile(); err != nil {
        return nil, err
    }
    if err := g.doIncludeRegexps(); err != nil {
        return nil, err
    }
    return g, nil
}

func (g *Game) includeFromFile() error { // TODO: It would be nice if this wouldnt overwrite anything that has been set
    if g.Include == "" {                 //       But doing that isnt exactly simple. For now it just overwrites stuff set
        return nil                       //       in the included file, ie, if you set something and its set in the included
    }                                    //       file, the stuff in the included file wins

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

func (g *Game) doIncludeRegexps() error {
    if g.IncludeRegexp == "" {
        return nil
    }

    data, err := readAllFromFile(g.IncludeRegexp)
    if err != nil {
        return err
    }

    var toAdd []GameRegexp
    if err := xml.Unmarshal(data, &toAdd); err != nil {
        return err
    }

    g.Regexps = append(g.Regexps, toAdd...)
    return nil
}
