package transformer

import (
	"image/color"
	"strings"
)

// Transformer refers to a string transformer. String Transformers convert messages from an intermediate format
// to a protocol specific format
type Transformer interface {
	// Transform takes a string in the intermediate format and converts it to its specific format.
	// When the implementation of Transformer does not support a given format type, it can either eat it entirely, or
	// use a pure ascii notation to indicate that it was there. For example: strike though could be replaced with ~~
	Transform(in string) string
	// MakeIntermediate takes a string in a Transformer specific format and converts it to the Intermediate format.
	// Any existing sentinels in the string SHOULD be escaped
	MakeIntermediate(in string) string
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
		repl = append(repl, v, SentinelString+string(k))
	}

	for col, v := range colourMap {
		repl = append(repl, v, EmitColour(col))
	}

	repl = append(repl, SentinelString, SSentinelString)

	return &SimpleTransformer{
		rplMap:   replaceMap,
		palette:  palette,
		colMap:   colourMap,
		replacer: strings.NewReplacer(repl...),
	}

}

func (s *SimpleTransformer) Transform(in string) string {
	return Map(in, s.rplMap, s.colourFn)
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

func (s *SimpleTransformer) MakeIntermediate(in string) string {
	return s.replacer.Replace(in)
}
