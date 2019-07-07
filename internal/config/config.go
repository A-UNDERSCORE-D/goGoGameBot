package config

import (
	"encoding/xml"
	"io/ioutil"
	"os"
)

// Config is the top level struct that GGGB's XML config file is unpacked into
type Config struct {
	XMLName     xml.Name     `xml:"bot"`
	Irc         BotConfig    `xml:"bot_config"`
	Permissions []Permission `xml:"permissions>permission"`
	Ignores     []string     `xml:"ignore_mask"`
	Strips      []string     `xml:"strip_mask"`
	GameManager GameManager
	ConfigPath  string `xml:"-"`
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

func getXMLConf(filename string) (*Config, error) {
	data, err := readAllFromFile(filename)
	if err != nil {
		return nil, err
	}

	conf := new(Config)

	if err = xml.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

// GetConfig parses the config found at the given path and returns it. If a read error occurs while parsing, it is returned
func GetConfig(filename string) (*Config, error) {
	conf, err := getXMLConf(filename)

	if err != nil {
		return nil, err
	}
	conf.ConfigPath = filename
	return conf, nil
}
