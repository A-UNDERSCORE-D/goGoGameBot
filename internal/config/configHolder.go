package config

import (
	"encoding/xml"
	"fmt"
)

type ConfigHolder struct {
	Type   string // the type of the config we're holding
	Config string // the config itself
}

// UnmarshalXML Implements the Unmarshaler interface in the XML library (with tweaks). Specifically
// this is designed to unmarshal a single attr while maintaining the content of the rest of the tag for
// later unmarshalinmg
func (c *ConfigHolder) UnmarshalXML(name, defaultType string, d *xml.Decoder, start xml.StartElement) error {
	if start.Name.Local != name {
		return nil
	}

	c.Type = defaultType

	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			c.Type = attr.Value
			break
		}
	}

	conf, err := reconstructXML(d, start)
	if err != nil {
		return fmt.Errorf("could not unmarshal XML for ConfigHolder(%s): %w", name, err)
	}

	c.Config = conf
	return nil
}
