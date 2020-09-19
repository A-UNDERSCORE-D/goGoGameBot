package config

// import (
// 	"encoding/xml"
// 	"fmt"
// )

// // ConfigHolder is a special construct that allows for partial unmarshaling of an XML object
// // Specifically, it is designed to allow configs to specify a type, and for code to change
// // objects used based on that type. Such as for Transport or Bot implementations
// type ConfigHolder_ struct {
// 	Type   string // the type of the config we're holding
// 	Config string // the config itself
// }

// // MagicUnmarshalXML Implements the Unmarshaler interface in the XML library (with tweaks). Specifically
// // this is designed to unmarshal a single attr while maintaining the content of the rest of the tag for
// // later unmarshaling
// func (c *ConfigHolder_) MagicUnmarshalXML(name, defaultType string, d *xml.Decoder, start xml.StartElement) error {
// 	if start.Name.Local != name {
// 		return nil
// 	}

// 	c.Type = defaultType

// 	for _, attr := range start.Attr {
// 		if attr.Name.Local == "type" {
// 			c.Type = attr.Value
// 			break
// 		}
// 	}

// 	conf, err := reconstructXML(d, start)
// 	if err != nil {
// 		return fmt.Errorf("could not unmarshal XML for ConfigHolder(%s): %w", name, err)
// 	}

// 	c.Config = conf
// 	return nil
// }
