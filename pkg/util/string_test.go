package util

import (
	"reflect"
	"testing"
)

var splitOnSpaceTests = []struct {
	name string
	args string
	want []string
}{
	{
		"basic",
		"test string is testy",
		[]string{"test", "string", "is", "testy"},
	},
	{
		"extra spaces",
		"test   string",
		[]string{"test", "string"},
	},
	{
		"no spaces",
		"teststring",
		[]string{"teststring"},
	},
	{
		"all spaces",
		"                ",
		[]string(nil),
	},
	{
		"no string",
		"",
		[]string(nil),
	},
}

func TestCleanSplitOnSpace(t *testing.T) {
	for _, tt := range splitOnSpaceTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanSplitOnSpace(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CleanSplitOnSpace() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func BenchmarkCleanSplitOnSpace(b *testing.B) {
	for _, tt := range splitOnSpaceTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				CleanSplitOnSpace(tt.args)
			}
		})
	}
}

type wordEolArgs struct {
	s       string
	wordIdx int
}

var wordEolTests = []struct {
	name string
	args wordEolArgs
	want string
}{
	{
		"simple",
		wordEolArgs{
			"test string",
			1,
		},
		"string",
	},
	{
		"bad idx",
		wordEolArgs{
			"test string",
			-1,
		},
		"",
	},
	{
		"empty string",
		wordEolArgs{
			"",
			1,
		},
		"",
	},
	{
		"return original",
		wordEolArgs{
			"test string",
			0,
		},
		"test string",
	},
	{
		"lots of spaces",
		wordEolArgs{
			"test                        string",
			0,
		},
		"test                        string",
	},
}

func TestWordEol(t *testing.T) {
	for _, tt := range wordEolTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WordEol(tt.args.s, tt.args.wordIdx); got != tt.want {
				t.Errorf("WordEol() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkWordEol(b *testing.B) {
	for _, tt := range wordEolTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				WordEol(tt.args.s, tt.args.wordIdx)
			}
		})
	}
}

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			"quote",
			`"`,
			`\"`,
		},
		{
			"nothing to escape",
			"test string",
			"test string",
		},
		{
			"random bytes",
			"#%f\x84\x88á\x0f\x84\x85¢£¦ëøâ¥rE»¤",
			"#%f\x84\x88á\x0f\x84\x85¢£¦ëøâ¥rE»¤",
		},
		{
			"multiple things that need escaping",
			`\!asd"''"some normal text / `,
			`\\!asd\"\'\'\"some normal text / `,
		},
		{
			"empty string",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeString(tt.args); got != tt.want {
				t.Errorf("EscapeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkEscapeString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EscapeString(`\!asd"''"some normal text / `)
	}
}
