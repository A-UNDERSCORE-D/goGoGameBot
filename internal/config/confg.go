package config

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"os"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// Config is the top level struct that GGGB's XML config file is unpacked into
type Config struct {
	XMLName     xml.Name     `xml:"bot"`
	Irc         BotConfig    `xml:"bot_config"`
	Permissions []Permission `xml:"permissions>permission"`
	Ignores     []string     `xml:"ignore_mask"`
	Strips      []string     `xml:"strip_mask"`
	Games       []GameConfig `xml:"game"`
}

var defaultConfig = Config{
	Irc: BotConfig{
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

	Games: []GameConfig{
		{
			Name:         "echo",
			AutoStart:    false,
			Path:         "/bin/echo",
			Args:         "test command is testy",
			LogChan:      "#ferricyanide",
			AdminLogChan: "#ferricyanide",

			Regexps: []GameRegexpConfig{{
				Name:      "test",
				Priority:  0,
				ShouldEat: true,
				Regexp:    "(.*)",
				Format:    util.Format{FormatString: "test"},
			}},
		},
	},
}

func readAllFromFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(f)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Config) runIncludes() error {
	for i, g := range c.Games {
		newG, err := g.doInclude()
		if err != nil {
			return err
		}
		c.Games[i] = *newG
	}
	return nil
}

func getXMLConf(filename string) (*Config, error) {
	data, err := readAllFromFile(filename)
	if err != nil {
		return nil, err
	}

	conf := new(Config)

	if err = xml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	if err = conf.runIncludes(); err != nil {
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

var ErrConfNotExist = errors.New("config file not found. Adding a default config")

// GetConfig parses the config found at the given path and returns it, if it does not exist it places the default config
// at that location, parses it, and returns that. If any other read error occurs while parsing, it is returned
func GetConfig(filename string) (*Config, error) {
	conf, err := getXMLConf(filename)

	if err != nil {
		if os.IsNotExist(err) {
			if err := writeDefaultConfig(filename); err != nil {
				return nil, err
			}

			return nil, ErrConfNotExist

		}
		return nil, err
	}

	return conf, nil
}
