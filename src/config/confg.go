package config

import (
    "encoding/xml"
    "io/ioutil"
    "os"
)

type IRC struct {
    Nick            string   `xml:"nick",json:"nick"`
    Ident           string   `xml:"ident",json:"ident"`
    Host            string   `xml:"host",json:"host"`
    Port            string   `xml:"port",json:"port"`
    SSL             bool     `xml:"ssl",json:"ssl"`
    ConnectCommands []string `xml:"connect_commands",json:"connect_commands"`
    JoinChans       []string `xml:"join_chans",json:"join_chans"`
    NSNick          string   `xml:"ns_nick",json:"ns_nick"`
    NSPasswd        string   `xml,json:"ns_passwd",json:"ns_passwd"`
}

type Game struct {
    Name      string
    AutoStart bool
}

type Config struct {
    Irc   IRC
    Games []Game
}

var defaultConfig Config = Config{
    Irc: IRC{
        Nick:            "goGoGameBot",
        Ident:           "GGGB",
        Host:            "irc.snoonet.org",
        Port:            "6697",
        SSL:             true,
        ConnectCommands: nil,
        JoinChans:       []string{"#ferricyanide"},
        NSNick:          "",
        NSPasswd:        "",
    },
    Games: nil,
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
    f, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := xml.MarshalIndent(defaultConfig, "", "    ")
    if err != nil {
        return err
    }

    if _, err = f.Write(data); err != nil {
        return err
    }

    return nil
}

//func getJSONConf(filename string) (*Config, error) {
//    f, err := os.Open(filename)
//    if err != nil {
//        return nil, err
//    }
//
//    var data []byte
//    if data, err = ioutil.ReadAll(f); err != nil {
//        return nil, err
//    }
//    conf := new(Config)
//
//    err = json.Unmarshal(data, conf)
//    if err != nil {
//        return nil, err
//    }
//
//    return conf, nil
//}

func GetConfig(filename string) (*Config, error) {
    conf, err := getXMLConf(filename)
    if err != nil && !os.IsNotExist(err) {
        return nil, err
    } else if err != nil && os.IsNotExist(err) {
        if err := writeDefaultConfig(filename); err != nil {
            return nil, err
        }
        return GetConfig(filename)
    }
    return conf, nil
}
