// Package minecraft contains a Transformer implementation for the JSON specification of minecraft format strings
// see https://minecraft.gamepedia.com/Raw_JSON_text_format for a full look at the format itself
package minecraft

import (
	"encoding/json"
	"image/color"
	"regexp"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/intermediate"
	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/tokeniser"
)

var colourMap = map[color.Color]string{
	black:       "black",
	darkBlue:    "dark_blue",
	darkGreen:   "dark_green",
	darkAqua:    "dark_aqua",
	darkRed:     "dark_red",
	darkPurple:  "dark_purple",
	gold:        "gold",
	gray:        "gray",
	darkGray:    "dark_gray",
	blue:        "blue",
	green:       "green",
	aqua:        "aqua",
	red:         "red",
	lightPurple: "light_purple",
	yellow:      "yellow",
	white:       "white",
}

var (
	black       = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF} // 0
	darkBlue    = color.RGBA{R: 0x00, G: 0x00, B: 0xAA, A: 0xFF} // 1
	darkGreen   = color.RGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0xFF} // 2
	darkAqua    = color.RGBA{R: 0x00, G: 0xAA, B: 0xAA, A: 0xFF} // 3
	darkRed     = color.RGBA{R: 0xAA, G: 0x00, B: 0x00, A: 0xFF} // 4
	darkPurple  = color.RGBA{R: 0xAA, G: 0x00, B: 0xAA, A: 0xFF} // 5
	gold        = color.RGBA{R: 0xFF, G: 0xAA, B: 0x00, A: 0xFF} // 6
	gray        = color.RGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xFF} // 7
	darkGray    = color.RGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xFF} // 8
	blue        = color.RGBA{R: 0x55, G: 0x55, B: 0xFF, A: 0xFF} // 9
	green       = color.RGBA{R: 0x55, G: 0xFF, B: 0x55, A: 0xFF} // a
	aqua        = color.RGBA{R: 0x55, G: 0xFF, B: 0xFF, A: 0xFF} // b
	red         = color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF} // c
	lightPurple = color.RGBA{R: 0xFF, G: 0x55, B: 0xFF, A: 0xFF} // d
	yellow      = color.RGBA{R: 0xFF, G: 0xFF, B: 0x55, A: 0xFF} // e
	white       = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // f
	palette     = color.Palette{
		black, darkBlue, darkGreen, darkAqua, darkRed, darkPurple,
		gold, gray, darkGray, blue, green, aqua, red, lightPurple,
		yellow, white,
	}
)

var urlRe = regexp.MustCompile(`https?://\S+\.\S+`)

type state struct {
	bold          bool
	italics       bool
	underline     bool
	strikethrough bool
	currentColour color.Color
	resetColour   bool
}

type clickEvent struct {
	Action string `json:"action"` // if we're a URL, this is open_url
	Value  string `json:"value"`
}

// TODO: this needs to be made to not omitEmpty because minecraft is dumb. Alternatively, an empty text element at the start
// 		 could work

type jsonSection struct {
	Text          string      `json:"text"` // The text to actually display
	Bold          bool        `json:"bold,omitempty"`
	Italic        bool        `json:"italic,omitempty"`
	Underline     bool        `json:"underlined,omitempty"`
	Strikethrough bool        `json:"strikethrough,omitempty"`
	Colour        string      `json:"color,omitempty"` // Why cant people spell colour?
	ClickEvent    *clickEvent `json:"clickEvent,omitempty"`
}

func (j *jsonSection) hasFormatting() bool {
	return j.Bold || j.Italic || j.Underline || j.Strikethrough || j.Colour != "" || j.ClickEvent != nil
}

func getJSONColour(s *state) string {
	if s.resetColour {
		s.resetColour = false
		return "reset"
	}

	if s.currentColour == nil {
		return ""
	}

	return colourMap[palette.Convert(s.currentColour)]
}

func jsonSectionFromState(str string, s *state, ce *clickEvent) jsonSection {
	return jsonSection{
		Text:          str,
		Bold:          s.bold,
		Italic:        s.italics,
		Underline:     s.underline,
		Strikethrough: s.strikethrough,
		Colour:        getJSONColour(s),
		ClickEvent:    ce,
	}
}

func urlState(s *state) *state {
	outState := *s
	outState.currentColour = blue
	outState.underline = true

	return &outState
}

func makeClickEvent(url string) *clickEvent {
	return &clickEvent{
		Action: "open_url",
		Value:  url,
	}
}

func splitOnURLs(in string, s *state) []jsonSection {
	URLLocations := urlRe.FindAllStringIndex(in, -1)
	if len(URLLocations) == 0 {
		return []jsonSection{jsonSectionFromState(in, s, nil)}
	}

	var (
		out    []jsonSection
		curIdx = 0
	)

	for _, idxPair := range URLLocations {
		url := in[idxPair[0]:idxPair[1]]
		curIdx = idxPair[1]

		prefix := in[curIdx:idxPair[0]]
		if len(prefix) > 0 {
			out = append(out, jsonSectionFromState(prefix, s, nil))
		}

		out = append(out, jsonSectionFromState(url, urlState(s), makeClickEvent(url)))
	}

	if curIdx < len(in)-1 {
		out = append(out, jsonSectionFromState(in[curIdx:], s, nil))
	}

	return out
}

// Transformer implements the transformer interface for Minecraft 1.8+ JSON formatting. The implementation is one way,
// ie, intermed -> minecraft. minecraft -> intermed simply escapes all sentinals
type Transformer struct{}

// Transform implements the transformer interface for Minecraft servers.
func (Transformer) Transform(in string) string {
	s := &state{}
	tokens := tokeniser.Tokenise(in)

	var out []jsonSection

	for _, tok := range tokens {
		switch tok.TokenType {
		case tokeniser.StringToken:
			out = append(out, splitOnURLs(tok.OriginalString, s)...)
		case intermediate.Colour:
			s.currentColour = tok.Colour
		case intermediate.Bold:
			s.bold = !s.bold
		case intermediate.Italic:
			s.italics = !s.italics
		case intermediate.Underline:
			s.underline = !s.underline
		case intermediate.Strikethrough:
			s.strikethrough = !s.strikethrough
		case intermediate.Reset:
			s.bold = false
			s.italics = false
			s.underline = false
			s.strikethrough = false
			s.currentColour = nil
			s.resetColour = true
		}
	}

	if len(out) > 1 && out[0].hasFormatting() {
		// Because MineCraft has some dumb as shit rules when it comes to first thing in a slice, lets (if we have to)
		// add an empty text value with no formatting that can be the default to avoid
		// "whoops everything has this formatting" issues
		out = append([]jsonSection{{}}, out...)
	}

	var (
		res []byte
		err error
	)

	switch len(out) {
	case 0:
		return ""
	case 1:
		res, err = json.Marshal(out[0])
	default:
		res, err = json.Marshal(out)
	}

	if err != nil {
		return "ERROR! " + err.Error()
	}

	return string(res)
}

// MakeIntermediate just returns the given strings escaped, as MineCraft servers dont output formatted strings ever
func (Transformer) MakeIntermediate(in string) string {
	return strings.Replace(in, intermediate.SentinelString, intermediate.SSentinelString, -1)
}
