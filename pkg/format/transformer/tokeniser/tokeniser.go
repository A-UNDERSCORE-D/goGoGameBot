package tokeniser

import (
	"image/color"
	"strings"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/format/transformer/intermediate"
)

// StringToken is the value given to Token instances holding a raw string
const StringToken = -1

// Token represents a single chunk of intermediate format information
type Token struct {
	TokenType      int
	Colour         color.Color
	OriginalString string
}

// Tokenise turns an input string containing intermediate format codes and returns a slice of Tokens representing
// the data given. It is intended for use by Transformer implementations that do not want to do the heavy lifting
// required to parse the intermediate format
func Tokenise(in string) []Token {
	var out []Token

	buf := strings.Builder{}
	seenSentinel := false
	skip := 0

	for i, r := range in {
		if skip > 0 {
			skip--
			continue
		}

		switch r {
		case intermediate.Sentinel:
			if seenSentinel || len(in) == i+1 {
				seenSentinel = false

				buf.WriteRune(r)

				break
			}

			seenSentinel = true

		case intermediate.Colour:
			if seenSentinel {
				seenSentinel = false
				if len(in)-i < 7 {
					// Dont have enough space -- write out as if we didnt see it
					seenSentinel = false

					buf.WriteString(intermediate.SColourString)

					continue
				}

				col, err := ParseColour(in[i+1:])
				if err != nil {
					buf.WriteString(intermediate.SColourString)

					continue
				}

				if buf.Len() > 0 {
					out = append(out, Token{TokenType: StringToken, Colour: nil, OriginalString: buf.String()})

					buf.Reset()
				}

				out = append(out, Token{TokenType: int(r), Colour: col, OriginalString: ""})
				skip += 6

				continue
			}

			fallthrough

		case intermediate.Bold, intermediate.Italic, intermediate.Underline, intermediate.Strikethrough, intermediate.Reset:
			if seenSentinel {
				seenSentinel = false

				if buf.Len() > 0 {
					out = append(out, Token{TokenType: StringToken, Colour: nil, OriginalString: buf.String()})

					buf.Reset()
				}

				out = append(out, Token{TokenType: int(r), Colour: nil, OriginalString: ""})

				continue
			}

			fallthrough

		default:
			if seenSentinel {
				// Invalid trailing char after sentinel
				buf.WriteRune(intermediate.Sentinel)
			}

			seenSentinel = false // Always reset this, just in case.

			buf.WriteRune(r)
		}
	}

	if buf.Len() > 0 {
		out = append(out, Token{TokenType: StringToken, Colour: nil, OriginalString: buf.String()})
	}

	return out
}
