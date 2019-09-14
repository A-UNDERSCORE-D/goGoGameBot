package transformer

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// Strip strips away all intermediate formatting
func Strip(in string) string {
	return Map(in, nil, nil)
}

func entryOrEmpty(r rune, mapping map[rune]string) string {
	if res, ok := mapping[r]; ok {
		return res
	}
	return ""
}

const alpha = 0xFF

// ParseColour Converts a string hex colour to a color.RGBA colour
func ParseColour(in string) (color.RGBA, error) {
	var r, g, b uint64
	var err error
	if r, err = strconv.ParseUint(in[0:2], 16, 8); err != nil {
		return color.RGBA{}, err
	}
	if g, err = strconv.ParseUint(in[2:4], 16, 8); err != nil {
		return color.RGBA{}, err
	}
	if b, err = strconv.ParseUint(in[4:6], 16, 8); err != nil {
		return color.RGBA{}, err
	}
	return color.RGBA{A: alpha, R: uint8(r), G: uint8(g), B: uint8(b)}, nil
}

// EmitColour is the companion to ParseColour, it converts a color.Color to the intermediate representation for
// use in larger tooling
func EmitColour(in color.Color) string {
	r, g, b, _ := in.RGBA()
	return fmt.Sprintf("$c%02X%02X%02X", uint8(r), uint8(g), uint8(b))
}

// Map maps a string containing intermediate formatting to the strings specified by the mapping arg. Its a helper method
// to easily implement simple swapping for a Transformer implementation. If colourFn returns false, the sentinel and
// colour rune are added to the string as they are
func Map(in string, mapping map[rune]string, fn func(color.Color) string) string {
	tokenised := Tokenise(in)
	out := strings.Builder{}
	for _, tok := range tokenised {
		switch tok.TokenType {
		case StringToken:
			out.WriteString(tok.OriginalString)
		case Bold, Italic, Underline, Strikethrough, Reset:
			out.WriteString(entryOrEmpty(rune(tok.TokenType), mapping))
		case Colour:
			if fn == nil {
				continue // eat colour if unsupported
			}
			out.WriteString(fn(tok.Colour))
		}
	}
	return out.String()
}
