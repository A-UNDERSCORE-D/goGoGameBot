package transformer

import (
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

type ColourFn func(in string) (string, bool)

// Map maps a string containing intermediate formatting to the strings specified by the mapping arg. Its a helper method
// to easily implement simple swapping for a Transformer implementation. If colourFn returns false, the sentinel and
// colour rune are added to the string as they are
func Map(in string, mapping map[rune]string, fn ColourFn) string {
	out := strings.Builder{}

	seenSentinel := false
	skip := 0
	for i, r := range in {
		if skip > 0 {
			skip--
			continue
		}
		if r != Sentinel && !seenSentinel {
			out.WriteRune(r)
			continue
		}
		if !seenSentinel {
			seenSentinel = true
			if len(in) == i+1 /*Seen a sentinel, but its right at the end*/ {
				out.WriteRune(Sentinel)
			}
			continue
		}

		switch r {
		case Sentinel:
			out.WriteRune(Sentinel)
		case Bold, Italic, Underline, Strikethrough, Reset:
			out.WriteString(entryOrEmpty(r, mapping))
		case Colour:
			runes := []rune(in)
			if len(runes[i:]) < 7 || fn == nil {
				// We saw a $c, but there's not enough space here to contain an entire colour, or we dont have a colour
				// mapping
				out.WriteRune(Sentinel)
				out.WriteRune(Colour)
				break
			}

			res, ok := fn(string(runes[i+1 : i+7]))
			if ok {
				skip += 6
				out.WriteString(res)
			} else {
				out.WriteRune(Sentinel)
				out.WriteRune(Colour)
			}

		default:
			out.WriteRune(Sentinel)
			out.WriteRune(r)
		}
		seenSentinel = false

	}
	return out.String()
}
