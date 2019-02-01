package config

import "encoding/xml"

type IrcChan struct {
    Name string `xml:"name,attr"`
    Key  string `xml:"key,attr,omitempty"`
}

// TODO: add requirelogin bool
type NSAuth struct {
    Nick     string `xml:"nick,attr"`
    Password string `xml:"password,attr"`
    SASL     bool   `xml:"sasl,attr"`
}

type BotConfig struct {
    XMLName         xml.Name  `xml:"bot_config"`
    Nick            string    `xml:"nick,attr"`
    Ident           string    `xml:"ident,attr"`
    Gecos           string    `xml:"gecos,attr"`
    Host            string    `xml:"host,attr"`
    Port            string    `xml:"port,attr"`
    SSL             bool      `xml:"ssl,attr"`
    CommandPrefix   string    `xml:"command_prefix,attr"`
    AdminChan       IrcChan   `xml:"admin_chan"`
    ConnectCommands []string  `xml:"connect_commands>command,omitempty"`
    JoinChans       []IrcChan `xml:"autojoin_channels>channel,omitempty"`
    NSAuth          NSAuth    `xml:"auth>nickserv"`
}

type Permission struct {
    XMLName xml.Name `xml:"permission"`
    Mask    string   `xml:"mask,attr"`
}
