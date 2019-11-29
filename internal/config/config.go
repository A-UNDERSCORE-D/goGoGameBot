package config

import (
	"encoding/xml"
	"io/ioutil"
)

// Config is the top level struct that GGGBs XML config file is unpacked into
type Config struct {
	XMLName     xml.Name   `xml:"bot"`
	ConnConfig  ConnConfig `xml:"conn_config"`
	GameManager GameManager
	ConfigPath  string `xml:"-"`
}

// ConnConfig represents a config for the connection named by ConnType. The config itself is a reconstructed XML stream
// which means that this type implements xml.Unmarshaler
type ConnConfig struct {
	ConfigHolder
}

// UnmarshalXML implements the unmarshaler interface in the XML package. In this instance, it takes the XML token stream
// and reconstructs it, storing the result in ConnConfig.Config, for later parsing by the named ConnType's constructor
func (c *ConnConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	return c.ConfigHolder.UnmarshalXML("conn_config", "", d, start)
	// if start.Name.Local != "conn_config" {
	// 	return nil
	// }
	// for _, attr := range start.Attr {
	// 	if attr.Name.Local == "conn_type" {
	// 		c.ConnType = attr.Value
	// 		break
	// 	}
	// }
	//
	// conf, err := reconstructXML(d, start)
	// if err != nil {
	// 	return fmt.Errorf("could not unmarshal XML for ConnConfig: %w", err)
	// }
	//
	// c.Config = conf
	// return nil
}

func readAllFromFile(name string) ([]byte, error) {
	data, err := ioutil.ReadFile(name)
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

	if err := xml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

// GetConfig parses the config found at the given path and returns it.
// If a read error occurs while parsing, it is returned
func GetConfig(filename string) (*Config, error) {
	conf, err := getXMLConf(filename)

	if err != nil {
		return nil, err
	}

	conf.ConfigPath = filename

	return conf, nil
}
