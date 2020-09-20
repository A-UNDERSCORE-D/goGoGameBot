package minecraft

import (
	"fmt"
	"strings"
	"testing"
)

var tests = []struct {
	name  string
	input string
	want  string
}{
	{
		name:  "plain message",
		input: "plain message",
		want:  `{"text":"plain message"}`,
	},
	{
		name:  "single sentinel",
		input: "$$",
		want:  `{"text":"$"}`,
	},
	{
		name:  "random interleaves",
		input: "this$b has$i some random $s chars put $b throughout $i it $u have fun $r with this",
		want: `[{"text":"this"},{"text":" has","bold":true},{"text":" some random ","bold":true,"italic":true},` +
			`{"text":" chars put ","bold":true,"italic":true,"strikethrough":true},` +
			`{"text":" throughout ","italic":true,"strikethrough":true},{"text":" it ","strikethrough":true},` +
			//nolint:misspell // minecraft devs cant spell colour either
			`{"text":" have fun ","underlined":true,"strikethrough":true},{"text":" with this","color":"reset"}]`,
	},
	{
		name:  "colour spam",
		input: "this $cFFFFFFhas a bunch of $c000000colours in it so$c012345 it can test $r colour barf",
		//nolint:misspell // minecraft devs cant spell colour either
		want: `[{"text":"this "},{"text":"has a bunch of ","color":"white"},` +
			`{"text":"colours in it so","color":"black"},{"text":" it can test ","color":"black"},` +
			`{"text":" colour barf","color":"reset"}]`,
	},
	{
		name:  "extra formatting",
		input: "A thing$b with$b$bsome weird$iformats$b$i$i and $b stuff",
		want: `[{"text":"A thing"},{"text":" with","bold":true},{"text":"some weird","bold":true},` +
			`{"text":"formats","bold":true,"italic":true},{"text":" and ","italic":true},` +
			`{"text":" stuff","bold":true,"italic":true}]`,
	},
	{
		name:  "URL test",
		input: "hey check out this thing! https://git.ferricyanide.solutions/A_D/goGoGameBot it has cool stuff!",
		want: `[{"text":"hey check out this thing! "},` +
			`{"text":"https://git.ferricyanide.solutions/A_D/goGoGameBot",` +
			`"underlined":true,"color":"blue","clickEvent":{"action":"open_url",` + //nolint:misspell // no choice
			`"value":"https://git.ferricyanide.solutions/A_D/goGoGameBot"}},{"text":" it has cool stuff!"}]`,
	},
	{
		name:  "url test 2",
		input: "https://google.com",
		want: `{"text":"https://google.com","underlined":true,"color":"blue",` +
			`"clickEvent":{"action":"open_url","value":"https://google.com"}}`,
	},
	{
		name:  "url test 3",
		input: "this is a test https://google.com",
		want: `[{"text":"this is a test "},` +
			`{"text":"https://google.com","underlined":true,"color":"blue","clickEvent":` +
			`{"action":"open_url","value":"https://google.com"}}]`,
	}, {
		name:  "url leader",
		input: "https://github.com/ this is a test",
		want: `[{"text":""},{"text":"https://github.com/","underlined":true,"color":"blue",` +
			`"clickEvent":{"action":"open_url","value":"https://github.com/"}},{"text":" this is a test"}]`,
	},
	{
		name:  "test using actual line",
		input: "$c666666$i[#totallynotgames]$rtest message",
		want: `[{"text":""},{"text":"[#totallynotgames]","italic":true,"color":"dark_gray"},` + //nolint:misspell // no choice
			`{"text":"test message","color":"reset"}]`, //nolint:misspell // no choice
	},
}

func findFirstDiff(s1, s2 string) {
	longer := s1
	shorter := s2

	if len(s1) < len(s2) {
		longer = s2
		shorter = s1
	}

	differencesStartAt := 0

	for i := 0; i < len(shorter); i++ {
		if shorter[i] != longer[i] {
			differencesStartAt = i
			break
		}
	}

	if differencesStartAt > 0 {
		fmt.Println("1:", shorter)
		fmt.Println("2:", longer)
		fmt.Println(strings.Repeat(" ", differencesStartAt) + "^")
	}
}

func TestTransformer_Transform(t *testing.T) {
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := (Transformer{}).Transform(tt.input); got != tt.want {
				t.Errorf("Transform(): got %s, want %s", got, tt.want)
				findFirstDiff(got, tt.want)
			}
		})
	}
}
