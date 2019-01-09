package config

import (
    "encoding/xml"
    "io/ioutil"
    "os"
)

type Config struct {
    XMLName     xml.Name     `xml:"bot"`
    Irc         IRC          `xml:"irc"`
    Permissions []Permission `xml:"permissions>permission"`
    Games       []Game       `xml:"game"`
}

var defaultConfig = Config{
    Irc: IRC{
        CommandPrefix:   "~",
        Nick:            "goGoGameBot",
        Ident:           "GGGB",
        Gecos:           "Go Game Manager",
        Host:            "irc.snoonet.org",
        Port:            "6697",
        SSL:             true,
        ConnectCommands: []string{"PING :Teststuff"},
        JoinChans:       []IrcChan{{Name: "#ferricyanide"}, {Name: "#someOtherChan"}},
        NSAuth:          NSAuth{"goGoGameBot", "goGoSuperSecurePasswd", true},
    },
    Permissions: []Permission{{Mask: "*!*@snoonet/staff/A_D"}},
    Games:       []Game{
        {
            Name:         "echo",
            AutoStart:    false,
            Path:         "/bin/echo",
            Args:         "test command is testy",
            Logchan:      "#ferricyanide",
            AdminLogChan: "#ferricyanide",
            Regexps:      []GameRegexp{{
                Name:      "test",
                Priority:  0,
                ShouldEat: true,
                Regexp:    "(.*)",
                Format:    "test",
            }},
        },
    },
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
