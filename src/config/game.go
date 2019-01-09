package config

import "encoding/xml"

type Game struct {
    XMLName      xml.Name `xml:"game"`
    Name         string   `xml:"name,attr"`
    AutoStart    bool     `xml:"auto_start,attr"`
    Path         string   `xml:"bin_path,attr"`
    Args         string   `xml:"args,attr"`
    Logchan      string   `xml:"log_chan,attr"`
    AdminLogChan string   `xml:"admin_log_chan,attr"`
}
