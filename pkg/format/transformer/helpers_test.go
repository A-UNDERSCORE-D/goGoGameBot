package transformer

import (
	"fmt"
	"image/color"
	"testing"
)

func TestMap(t *testing.T) {
	defaultMap := map[rune]string{
		Bold:          "bold",
		Italic:        "italic",
		Underline:     "underline",
		Strikethrough: "strikethrough",
		Reset:         "reset",
	}
	cFn := ColourFn(func(in string) (string, bool) {
		c, err := ParseColour(in)
		if err != nil {
			return string(Sentinel) + string(Colour), false // Return $c because that's all that would be eaten otherwise
		}

		return fmt.Sprintf("COLOUR[%0X%0X%0X]", c.R, c.G, c.B), true

	})

	tests := []struct {
		name    string
		in      string
		mapping map[rune]string
		want    string
	}{
		{
			name:    "non-special string",
			in:      "this is a test",
			mapping: defaultMap,
			want:    "this is a test",
		},
		{
			name:    "non-special with sentinel",
			in:      "thi$ is a te$t string",
			mapping: defaultMap,
			want:    "thi$ is a te$t string",
		},
		{
			name:    "escaped sentinel",
			in:      "this$$ $$ is a tes$$t string",
			mapping: defaultMap,
			want:    "this$ $ is a tes$t string",
		},
		{
			name:    "various magic chars",
			in:      "this is a $btest$r of $ivarious$u characters, $sbutNotAll$r",
			mapping: defaultMap,
			want:    "this is a boldtestreset of italicvariousunderline characters, strikethroughbutNotAllreset",
		},
		{
			name:    "sentinel at beginning and end",
			in:      "$ this is a test with an end sentinel$",
			mapping: defaultMap,
			want:    "$ this is a test with an end sentinel$",
		},
		{
			name:    "nill mapping",
			in:      "test",
			mapping: nil,
			want:    "test",
		},
		{
			name:    "fakeout colour",
			in:      "this $cmeep is a test colour",
			mapping: defaultMap,
			want:    "this $cmeep is a test colour",
		},
		{
			name:    "real colour",
			in:      "this test has $cAABBCC colours in it$c",
			mapping: defaultMap,
			want:    "this test has COLOUR[AABBCC] colours in it$c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Map(tt.in, tt.mapping, cFn); got != tt.want {
				t.Errorf("Map() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "non-special string",
			in:   "this is a test",
			want: "this is a test",
		},
		{
			name: "non-special with sentinel",
			in:   "thi$ is a te$t string",
			want: "thi$ is a te$t string",
		},
		{
			name: "escaped sentinel",
			in:   "this$$ $$ is a tes$$t string",
			want: "this$ $ is a tes$t string",
		},
		{
			name: "various magic chars",
			in:   "this is a $btest$r of $ivarious$u characters, $sbutNotAll$r",
			want: "this is a test of various characters, butNotAll",
		},
		{
			name: "sentinel at beginning and end",
			in:   "$ this is a test with an end sentinel$",
			want: "$ this is a test with an end sentinel$",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Strip(tt.in); got != tt.want {
				t.Errorf("Strip() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmitColour(t *testing.T) {
	tests := []struct {
		name string
		in   color.Color
		want string
	}{
		{
			"all FF",
			color.RGBA{R: 255, G: 255, B: 255, A: 255},
			"$cFFFFFF",
		},
		{
			"all 0",
			color.RGBA{R: 0, G: 0, B: 0, A: 0},
			"$c000000",
		},
		{
			"133337",
			color.RGBA{R: 13, G: 33, B: 37, A: 0},
			"$c0D2125",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EmitColour(tt.in); got != tt.want {
				t.Errorf("EmitColour() = %q, want %q", got, tt.want)
			}
		})
	}
}
