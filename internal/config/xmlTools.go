package config

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func reconstructXML(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	buf := bytes.NewBuffer(tokenToBytes(start))
	for {
		t, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("could not reconstruct XML: %w", err)
		}

		buf.Write(tokenToBytes(t))
		if t == start.End() {
			break
		}
	}
	return buf.String(), nil
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
