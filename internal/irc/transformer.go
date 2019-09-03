package irc

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"unicode"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer"
)

const (
	bold      = '\x02'
	italic    = '\x1d'
	underline = '\x1F'
	reset     = '\x0F'
	colour    = '\x03'
)

var (
	white      = color.RGBA{A: 255, R: 255, G: 255, B: 255} // 00
	black      = color.RGBA{A: 255, R: 0, G: 0, B: 0}       // 01
	blue       = color.RGBA{A: 255, R: 0, G: 0, B: 127}     // 02
	green      = color.RGBA{A: 255, R: 0, G: 147, B: 0}     // 03
	lightRed   = color.RGBA{A: 255, R: 255, G: 0, B: 0}     // 04
	brown      = color.RGBA{A: 255, R: 127, G: 0, B: 0}     // 05
	purple     = color.RGBA{A: 255, R: 156, G: 0, B: 156}   // 06
	orange     = color.RGBA{A: 255, R: 252, G: 127, B: 0}   // 07
	yellow     = color.RGBA{A: 255, R: 255, G: 255, B: 0}   // 08
	lightGreen = color.RGBA{A: 255, R: 0, G: 252, B: 0}     // 09
	cyan       = color.RGBA{A: 255, R: 0, G: 147, B: 147}   // 10
	lightCyan  = color.RGBA{A: 255, R: 0, G: 255, B: 255}   // 11
	lightBlue  = color.RGBA{A: 255, R: 0, G: 0, B: 252}     // 12
	pink       = color.RGBA{A: 255, R: 255, G: 0, B: 255}   // 13
	grey       = color.RGBA{A: 255, R: 127, G: 127, B: 127} // 14
	lightGrey  = color.RGBA{A: 255, R: 210, G: 210, B: 210} // 15
)

var ircPalette = color.Palette{
	white, black, blue, green, lightRed, brown, purple, orange, yellow,
	lightGreen, cyan, lightCyan, lightBlue, pink, grey, lightGrey,
}

var ircFmtMapping = map[rune]string{
	transformer.Bold:      string(bold),
	transformer.Italic:    string(italic),
	transformer.Underline: string(underline),
	transformer.Reset:     string(reset),
}

func reverseLookupMap(r rune) rune {
	for k, v := range ircFmtMapping {
		if string(r) == v {
			return k
		}
	}
	return 0
}

// Transformer is a dummy struct that holds methods for IRC's implementation of format/transformer's transformer interface
type Transformer struct{}

func (Transformer) Transform(in string) string {
	return transformer.Map(in, ircFmtMapping, func(in string) (s string, b bool) {
		if c, err := transformer.ParseColour(in); err != nil {
			return "", false
		} else {
			return fmt.Sprintf("%c%02d", colour, ircPalette.Index(c)), true
		}
	})
}

func (Transformer) MakeIntermediate(in string) string {
	out := strings.Builder{}
	skip := 0
	for i, r := range in {
		if skip > 0 {
			skip--
			continue
		}

		switch r {
		case bold, italic, underline, reset:
			out.WriteRune(transformer.Sentinel)
			out.WriteRune(reverseLookupMap(r))
		case transformer.Sentinel:
			out.Write([]byte{transformer.Sentinel, transformer.Sentinel})
		case colour:
			toSkip, col := extractColour(in[i:])
			if toSkip != -1 {
				skip += toSkip
				out.WriteString(transformer.EmitColour(col))
			}

		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

// extractColour returns the colour found at the beginning of the given string, and returns either the colour and
// the number of runes to skip, or -1 and nil
func extractColour(in string) (int, color.Color) {
	if len(in) < 1 || in[0] != colour {
		return -1, nil
	}

	fg := strings.Builder{}
	c := 0
	seenComma := false
	for i, r := range in[1:] {
		if !unicode.IsDigit(r) && !(r == ',' && i == 2) {
			if seenComma {
				c--
			}
			break
		} else if r == ',' && i == 2 {
			seenComma = true
		}

		if i == 2 && !seenComma {
			break
		}

		if c < 2 {
			fg.WriteRune(r)
		}
		c++
		if c == 5 {
			break
		}
	}
	idx, err := strconv.Atoi(fg.String())

	if err != nil || (idx > 15 || idx < 0) {
		return -1, nil
	}

	return c, ircPalette[idx]
}
