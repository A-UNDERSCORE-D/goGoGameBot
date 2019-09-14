package transformer

import (
	"image/color"
	"reflect"
	"testing"
)

func TestTokenise(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Token
	}{
		{
			name: "simple",
			in:   "this is a test",
			want: []Token{{TokenType: StringToken, Colour: nil, OriginalString: "this is a test"}},
		},
		{
			name: "no colours",
			in:   "this is a $btest$r that has some $scodes in it$r",
			want: []Token{
				{TokenType: StringToken, OriginalString: "this is a "},
				{TokenType: Bold},
				{TokenType: StringToken, OriginalString: "test"},
				{TokenType: Reset},
				{TokenType: StringToken, OriginalString: " that has some "},
				{TokenType: Strikethrough},
				{TokenType: StringToken, OriginalString: "codes in it"},
				{TokenType: Reset},
			},
		},
		{
			name: "empty",
			in:   "",
			want: nil,
		},
		{
			name: "single sentinel",
			in:   "$",
			want: []Token{{TokenType: StringToken, OriginalString: "$"}},
		},
		{
			name: "double sentinel",
			in:   "$$",
			want: []Token{{TokenType: StringToken, OriginalString: "$"}},
		},
		{
			name: "colour test",
			in:   "colours $cFFFFFF are fun",
			want: []Token{
				{TokenType: StringToken, OriginalString: "colours "},
				{TokenType: Colour, Colour: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}},
				{TokenType: StringToken, OriginalString: " are fun"},
			},
		},
		{
			name: "bad colour",
			in:   "$clolhavesomepadding",
			want: []Token{{TokenType: StringToken, OriginalString: "$clolhavesomepadding"}},
		},
		{
			name: "interspersed sentinels",
			in:   "thi$ is a te$t string",
			want: []Token{{TokenType: StringToken, OriginalString: "thi$ is a te$t string"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tokenise(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenise() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
