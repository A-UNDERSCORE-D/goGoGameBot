package simple

import (
	"image/color"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/intermediate"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/tokeniser"
)

// Conf Holds a replacemap and a colourmap in a format that's simple to store in XML
type Conf struct {
	ReplaceMap struct {
		Bold          string `xml:"bold"`
		Italic        string `xml:"italic"`
		Underline     string `xml:"underline"`
		Strikethrough string `xml:"strikethrough"`
		Reset         string `xml:"reset"`
	} `xml:"replace_map"`

	ColourMap []struct {
		R      uint8  `xml:"r,attr"`
		G      uint8  `xml:"g,attr"`
		B      uint8  `xml:"b,attr"`
		Mapped string `xml:",chardata"`
	} `xml:"colour_map>colour"`
}

func (s *Conf) MakeMaps() (map[rune]string, map[color.Color]string) {
	replaceMap := map[rune]string{
		intermediate.Bold:          s.ReplaceMap.Bold,
		intermediate.Italic:        s.ReplaceMap.Italic,
		intermediate.Underline:     s.ReplaceMap.Underline,
		intermediate.Strikethrough: s.ReplaceMap.Strikethrough,
		intermediate.Reset:         s.ReplaceMap.Reset,
	}
	colourMap := make(map[color.Color]string)
	for _, cc := range s.ColourMap {
		c := color.RGBA{
			R: cc.R,
			G: cc.G,
			B: cc.B,
			A: 0xFF,
		}
		colourMap[c] = cc.Mapped
	}
	return replaceMap, colourMap
}

// SimpleTransformer is a Transformer implementation that does basic replacement based transformation.
// Colours are handled by way of a palette and a map to transform colours in that palette to the transformer specific
// format
type SimpleTransformer struct {
	rplMap   map[rune]string
	palette  color.Palette
	colMap   map[color.Color]string
	replacer *strings.Replacer
}

// NewSimpleTransformer constructs a SimpleTransformer from the given args. A colour palette will be automatically
// created from the colour map passed.
func NewSimpleTransformer(replaceMap map[rune]string, colourMap map[color.Color]string) *SimpleTransformer {
	var palette color.Palette
	for col := range colourMap {
		palette = append(palette, col)
	}

	var repl []string
	for k, v := range replaceMap {
		repl = append(repl, v, intermediate.SentinelString+string(k))
	}

	for col, v := range colourMap {
		repl = append(repl, v, tokeniser.EmitColour(col))
	}

	repl = append(repl, intermediate.SentinelString, intermediate.SSentinelString)

	return &SimpleTransformer{
		rplMap:   replaceMap,
		palette:  palette,
		colMap:   colourMap,
		replacer: strings.NewReplacer(repl...), // the repl slice is reversed from the map* maps, this way it does an inverse
	}

}

// Transform implements the Transformer interface. Applies the simple conversions setup in the constructor
func (s *SimpleTransformer) Transform(in string) string {
	return tokeniser.Map(in, s.rplMap, s.colourFn)
}

func (s *SimpleTransformer) colourFn(in color.Color) string {
	if s.palette == nil || len(s.palette) == 0 {
		return ""
	}

	return s.colMap[s.palette.Convert(in)]
}

func (s *SimpleTransformer) reverseColour(in string) color.Color {
	for c, s := range s.colMap {
		if s == in {
			return c
		}
	}
	return nil
}

// MakeIntermediate uses a simple replace operation to convert from a transformer specific implementation to the
// intermediate format
func (s *SimpleTransformer) MakeIntermediate(in string) string {
	return s.replacer.Replace(in)
}