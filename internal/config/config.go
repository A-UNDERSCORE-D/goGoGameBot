package config

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"os"
)

// Config is the top level struct that GGGB's XML config file is unpacked into
type Config struct {
	XMLName     xml.Name   `xml:"bot"`
	ConnConfig  ConnConfig `xml:"conn_config"`
	GameManager GameManager
	ConfigPath  string `xml:"-"`
}

// ConnConfig represents a config for the connection named by ConnType. The config itself is a reconstructed XML stream
// which means that this type implements xml.Unmarshaler
type ConnConfig struct {
	ConnType string
	Config   string
}

// UnmarshalXML implements the unmarshaler interface in the XML package. In this instance, it takes the XML token stream
// and reconstructs it, storing the result in ConnConfig.Config, for later parsing by the named ConnType's constructor
func (c *ConnConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if start.Name.Local != "conn_config" {
		return nil
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "conn_type" {
			c.ConnType = attr.Value
			break
		}
	}
	buf := bytes.NewBuffer(tokenToBytes(start))
	for {
		t, err := d.Token()
		if err != nil {
			return err
		}
		buf.Write(tokenToBytes(t))
		if t == start.End() {
			break
		}

	}

	c.Config = buf.String()
	return nil
}

func nameToBytes(name xml.Name) (out []byte) {
	if name.Space != "" {
		out = append(out, name.Space...)
		out = append(out, ':')
	}
	out = append(out, name.Local...)
	return
}

func attrToBytes(a xml.Attr) []byte {
	if a.Name.Space != "" {
		// For us this is fine because we never use name spacing, and we cant
		// reconstruct this easily / at all because of how its parsed out
		return nil
	}
	out := bytes.Buffer{}
	out.Write(nameToBytes(a.Name))
	out.WriteRune('=')
	out.WriteRune('"')
	out.WriteString(a.Value)
	out.WriteRune('"')
	return out.Bytes()
}

func tokenToBytes(t xml.Token) []byte {
	buf := bytes.Buffer{}
	switch t.(type) {
	case xml.StartElement:
		se := t.(xml.StartElement)
		buf.WriteRune('<')
		if se.Name.Space != "" {
			buf.WriteString(se.Name.Space)
			buf.WriteRune(':')
		}

		buf.WriteString(se.Name.Local)
		for _, v := range se.Attr {
			buf.WriteRune(' ')
			buf.Write(attrToBytes(v))
		}

		buf.WriteRune('>')

	case xml.EndElement:
		ee := t.(xml.EndElement)
		buf.WriteString("</")
		buf.Write(nameToBytes(ee.Name))
		buf.WriteRune('>')

	case xml.CharData:
		buf.Write(t.(xml.CharData))
	case xml.Comment:
		buf.WriteString("<!-- ")
		buf.Write(t.(xml.Comment))
		buf.WriteString(" -->")

	default:
		panic("unexpected token in XML stream")
	}

	return buf.Bytes()
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
