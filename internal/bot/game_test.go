package bot

import (
	"fmt"
	"testing"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"

	"git.ferricyanide.solutions/A_D/goGoGameBot/internal/config"
)

func TestGame_MapColours(t *testing.T) {
	var tests = [][]string{
		{"test string", "test string"},
		{"\x0304t\x0307e\x0308s\x0309t \x0310s\x0312t\x0302r\x0306i\x0313n\x0305g", "RedtOrangeeYellowsLightGreent CyansLightBluetBluerMagentaiPinknBrowng"},
		{"\x034$", "Red$"},
		{"$c", "$c"},
		{"$c[]", "$c[]"},
		{"$c[red]", "$c[red]"},
	}
	_ = tests
	colourMapRaw := config.ColourMap{
		Bold:          "Bold",
		Italic:        "Italic",
		ReverseColour: "ReverseColour",
		Strikethrough: "Strikethrough",
		Underline:     "Underline",
		Monospace:     "Monospace",
		Reset:         "Reset",
		White:         "White",
		Black:         "Black",
		Blue:          "Blue",
		Green:         "Green",
		Red:           "Red",
		Brown:         "Brown",
		Magenta:       "Magenta",
		Orange:        "Orange",
		Yellow:        "Yellow",
		LightGreen:    "LightGreen",
		Cyan:          "Cyan",
		LightCyan:     "LightCyan",
		LightBlue:     "LightBlue",
		Pink:          "Pink",
		Grey:          "Grey",
		LightGrey:     "LightGrey",
	}
	cmap, err := util.MakeColourMap(colourMapRaw.ToMap())
	if err != nil {
		t.Error(err)
	}
	testGame := Game{colourMap: cmap}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%q", test[1]), func(t *testing.T) {
			res := testGame.MapColours(test[0])
			if res != test[1] {
				t.Errorf("for: %q expected: %q got %q", test[0], test[1], res)
			}
		})
	}
}
