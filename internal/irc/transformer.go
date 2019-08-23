package irc

import (
	"fmt"
	"image/color"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/formatTransformer"
)

const (
	bold      = '\x02'
	italic    = '\x1d'
	underline = '\x1F'
	reset     = '\x0F'
	colour    = '\x03'
)

var (
	white      = color.RGBA{A: 255, R: 255, G: 255, B: 255}
	black      = color.RGBA{A: 255, R: 0, G: 0, B: 0}
	blue       = color.RGBA{A: 255, R: 0, G: 0, B: 127}
	green      = color.RGBA{A: 255, R: 0, G: 147, B: 0}
	lightRed   = color.RGBA{A: 255, R: 255, G: 0, B: 0}
	brown      = color.RGBA{A: 255, R: 127, G: 0, B: 0}
	purple     = color.RGBA{A: 255, R: 156, G: 0, B: 156}
	orange     = color.RGBA{A: 255, R: 252, G: 127, B: 0}
	yellow     = color.RGBA{A: 255, R: 255, G: 255, B: 0}
	lightGreen = color.RGBA{A: 255, R: 0, G: 252, B: 0}
	cyan       = color.RGBA{A: 255, R: 0, G: 147, B: 147}
	lightCyan  = color.RGBA{A: 255, R: 0, G: 255, B: 255}
	lightBlue  = color.RGBA{A: 255, R: 0, G: 0, B: 252}
	pink       = color.RGBA{A: 255, R: 255, G: 0, B: 255}
	grey       = color.RGBA{A: 255, R: 127, G: 127, B: 127}
	lightGrey  = color.RGBA{A: 255, R: 210, G: 210, B: 210}
)

var ircPalette = color.Palette{
	white, black, blue, green, lightRed, brown, purple, orange, yellow,
	lightGreen, cyan, lightCyan, lightBlue, pink, grey, lightGrey,
}

var ircFmtMapping = map[rune]string{
	formatTransformer.Bold:      string(bold),
	formatTransformer.Italic:    string(italic),
	formatTransformer.Underline: string(underline),
	formatTransformer.Reset:     string(reset),
}

func reverseLookupMap(r rune) rune {
	for k, v := range ircFmtMapping {
		if string(r) == v {
			return k
		}
	}
	return 0
}

type IRCTransformer struct{}

func (IRCTransformer) Transform(in string) string {
	return formatTransformer.Map(in, ircFmtMapping, func(in string) (s string, b bool) {
		if c, err := formatTransformer.ParseColour(in); err != nil {
			return "", false
		} else {
			return fmt.Sprintf("%b%d", colour, ircPalette.Index(c)), true
		}
	})
}

func (IRCTransformer) MakeIntermediate(in string) string {
	out := strings.Builder{}
	for _, r := range in {
		switch r {
		case bold, italic, underline, reset:
			out.WriteRune(formatTransformer.Sentinel)
			out.WriteRune(reverseLookupMap(r))
		case formatTransformer.Sentinel:
			out.Write([]byte{formatTransformer.Sentinel, formatTransformer.Sentinel})
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
