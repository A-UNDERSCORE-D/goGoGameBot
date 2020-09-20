// Package simple implements a Transformer that supports basic replacement based transformations
package simple

import (
	"image/color" //nolint:misspell // I dont control others' package names
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/intermediate"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/tokeniser"
)

// Conf Holds a replacemap and a colourmap in a format that's simple to store in XML
type Conf struct {
	ReplaceMap struct {
		Bold          string
		Italic        string
		Underline     string
		Strikethrough string
		Reset         string
	} `xml:"replace_map" comment:"Replace the listed formatting codes with the given string"`

	ColourMap []struct {
		R      uint8 `toml:"red"`
		G      uint8 `toml:"green"`
		B      uint8 `toml:"blue"`
		Mapped string
	} `toml:"map_colour" comment:"maps the given RGB colour to a string"`
}

// MakeMaps creates a replace and colour map based on the given config
func (s *Conf) MakeMaps() (replaceMap map[rune]string, colourMap map[color.Color]string) {
	replaceMap = map[rune]string{
		intermediate.Bold:          s.ReplaceMap.Bold,
		intermediate.Italic:        s.ReplaceMap.Italic,
		intermediate.Underline:     s.ReplaceMap.Underline,
		intermediate.Strikethrough: s.ReplaceMap.Strikethrough,
		intermediate.Reset:         s.ReplaceMap.Reset,
	}

	colourMap = make(map[color.Color]string)

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

// Transformer is a Transformer implementation that does basic replacement based transformation.
// Colours are handled by way of a palette and a map to transform colours in that palette to the transformer specific
// format
type Transformer struct {
	rplMap   map[rune]string
	palette  color.Palette
	colMap   map[color.Color]string
	replacer *strings.Replacer
}

// New constructs a Transformer from the given args. A colour palette will be automatically
// created from the colour map passed.
func New(replaceMap map[rune]string, colourMap map[color.Color]string) *Transformer {
	var palette color.Palette
	for col := range colourMap {
		palette = append(palette, col)
	}

	var repl []string

	for k, v := range replaceMap {
		if v == "" {
			continue
		}

		repl = append(repl, v, intermediate.SentinelString+string(k))
	}

	for col, v := range colourMap {
		repl = append(repl, v, tokeniser.EmitColour(col))
	}

	repl = append(repl, intermediate.SentinelString, intermediate.SSentinelString)

	return &Transformer{
		rplMap:   replaceMap,
		palette:  palette,
		colMap:   colourMap,
		replacer: strings.NewReplacer(repl...), // the repl slice is reversed from the map* maps, this way it does an inverse
	}
}

// Transform implements the Transformer interface. Applies the simple conversions setup in the constructor
func (s *Transformer) Transform(in string) string {
	return tokeniser.Map(in, s.rplMap, s.colourFn)
}

func (s *Transformer) colourFn(in color.Color) string {
	if s.palette == nil || len(s.palette) == 0 {
		return ""
	}

	return s.colMap[s.palette.Convert(in)]
}

// MakeIntermediate uses a simple replace operation to convert from a transformer specific implementation to the
// intermediate format
func (s *Transformer) MakeIntermediate(in string) string {
	return s.replacer.Replace(in)
}
