package config

import (
    "encoding/xml"
    "io/ioutil"
    "os"
)

type IrcChan struct {
    Name string `xml:",attr"`
    Key  string `xml:",attr,omitempty"`
}

// TODO: add requirelogin bool
type NSAuth struct {
    Nick     string `xml:"nick,attr"`
    Password string `xml:"password,attr"`
    SASL     bool   `xml:"sasl,attr"`
}

type IRC struct {
    XMLName         xml.Name  `xml:"irc"`
    Nick            string    `xml:"nick"`
    Ident           string    `xml:"ident"`
    Gecos           string    `xml:"gecos"`
    Host            string    `xml:"host,attr"`
    Port            string    `xml:"port,attr"`
    SSL             bool      `xml:"ssl,attr"`
    ConnectCommands []string  `xml:"connect_commands>command,omitempty"`
    JoinChans       []IrcChan `xml:"autojoin_channels>channel,omitempty"`
    NSAuth          NSAuth    `xml:"auth>nickserv"`
    CommandPrefix   string    `xml:"command_prefix,attr"`
}

type Game struct {
    XMLName   xml.Name `xml:"game"`
    Name      string   `xml:"name,attr"`
    AutoStart bool     `xml:"auto_start,attr"`
}

type Permission struct {
    XMLName xml.Name `xml:"permission"`
    Mask    string   `xml:"mask,attr"`
}

type Config struct {
    XMLName     xml.Name     `xml:"bot"`
    Irc         IRC          `xml:"irc"`
    Permissions []Permission `xml:"permissions>permission"`
    Games       []Game       `xml:"games>game"`
}

var defaultConfig Config = Config{
    Irc: IRC{
        CommandPrefix:   "~",
        Nick:            "goGoGameBot",
        Ident:           "GGGB",
        Gecos:           "Go Game Manager",
        Host:            "irc.snoonet.org",
        Port:            "6697",
        SSL:             true,
        ConnectCommands: []string{},
        JoinChans:       []IrcChan{{"#ferricyanide", ""}, {"#someOtherChan", ""}},
        NSAuth:          NSAuth{"goGoGameBot", "goGoSuperSecurePasswd", true},
    },
    Permissions: []Permission{{Mask: "*!*@snoonet/staff/A_D"}},
    Games:       nil,
}

func getXMLConf(filename string) (*Config, error) {
    f, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    data, err := ioutil.ReadAll(f)
    f.Close()
    if err != nil {
        return nil, err
    }

    conf := new(Config)

    err = xml.Unmarshal(data, conf)
    if err != nil {
        return nil, err
    }
    return conf, nil
}

func writeDefaultConfig(filename string) error {
    data, err := xml.MarshalIndent(defaultConfig, "", "    ")
    if err != nil {
        return err
    }

    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    if _, err = f.Write(data); err != nil {
        return err
    }

    return nil
}

func GetConfig(filename string) (*Config, error) {
    conf, err := getXMLConf(filename)

    if err != nil {
        if os.IsNotExist(err) {
            if err := writeDefaultConfig(filename); err != nil {
                return nil, err
            }

            return GetConfig(filename)
        } else {
            return nil, err
        }
    }

    return conf, nil
}
