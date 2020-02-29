package util

import (
	"fmt"
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
		{
			"newlines and co",
			"this is\n a test\r\n",
			"this is: a test:",
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

func TestStripAll(t *testing.T) {
	tests := []struct {
		name     string
		stringIn string
		want     string
	}{
		{"normal", "test string", "test string"},
		{"empty", "", ""},
		{"control codes", "test \x01 string \x02", "test  string "},
		{"zwsp", "test s\u200btring", "test string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripAll(tt.stringIn); got != tt.want {
				t.Errorf("StripAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ExampleStripAll() {
	fmt.Println(StripAll("Some string with \x01 control characters\u200b in it"))
	// output:
	// Some string with  control characters in it
}

func TestIdxOrEmpty(t *testing.T) {
	type args struct {
		slice []string
		idx   int
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"good", args{[]string{"test", "string"}, 1}, "string"},
		{"bad", args{[]string{"test", "string"}, 1337}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IdxOrEmpty(tt.args.slice, tt.args.idx); got != tt.want {
				t.Errorf("IdxOrEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ExampleIdxOrEmpty() {
	s := []string{"test", "string", "is", "testy"}

	fmt.Printf("%q\n", IdxOrEmpty(s, 0))
	fmt.Printf("%q\n", IdxOrEmpty(s, 5))
	//nolint:gocritic // its output :P
	// output:
	// "test"
	// ""
}

func TestJoinToMaxLength(t *testing.T) { //nolint:funlen // it contains test data
	type args struct {
		toJoin    []string
		sep       string
		maxLength int
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"normal",
			args{
				toJoin:    []string{"this", "is", "a", "test"},
				sep:       ", ",
				maxLength: 10,
			},
			[]string{"this, is", "a, test"},
		},
		{
			"constrained",
			args{
				toJoin:    []string{"this", "is", "a", "test"},
				sep:       ", ",
				maxLength: 1,
			},
			[]string{"this", "is", "a", "test"},
		},
		{
			"wide",
			args{
				toJoin:    []string{"this", "is", "a", "test"},
				sep:       ", ",
				maxLength: 100,
			},
			[]string{"this, is, a, test"},
		},
		{
			"empty",
			args{
				toJoin:    []string{""},
				sep:       ", ",
				maxLength: 100,
			},
			[]string(nil),
		},
		{
			"no split",
			args{
				toJoin:    []string{"this is a test"},
				sep:       ", ",
				maxLength: 1,
			},
			[]string{"this is a test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinToMaxLength(tt.args.toJoin, tt.args.sep, tt.args.maxLength); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("JoinToMaxLength() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func ExampleJoinToMaxLength() {
	fmt.Printf("%#v", JoinToMaxLength([]string{"this", "is", "a", "test"}, ", ", 10))
	// output:
	// []string{"this, is", "a, test"}
}

func TestReverseIdx(t *testing.T) {
	testArr := []string{
		"He", "just", "kept", "talking", "in", "one", "long", "incredibly", "unbroken", "sentence", "moving", "from",
		"topic", "so", "that", "no", "one", "had", "the", "chance", "to", "interrupt", "him",
	}
	tests := []struct {
		name string
		idx  int
		want string
	}{
		{"zero", 0, "He"},
		{"negative zero", -0, "He"},
		{"forward", 1, "just"},
		{"backward", -1, "him"},
		{"out of bounds", 1337, ""},
		{"inverse out of bounds", -1337, ""},
		{"middle idx", 7, "incredibly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReverseIdx(testArr, tt.idx); got != tt.want {
				t.Errorf("ReverseIdx() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ExampleReverseIdx() {
	fmt.Println(ReverseIdx([]string{"have", "an", "example", "string", "slice"}, -2))
	// output:
	// string
}

var addZwspTests = []struct {
	name string
	args string
	want string
}{
	{"empty string", "", ""},
	{"test string", "A_Dragon", "A\u200b_Dragon"},
	{"short test", "AD", "A\u200bD"},
	{"one char", "A", "A"},
}

func TestAddZwsp(t *testing.T) {
	for _, tt := range addZwspTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddZwsp(tt.args); got != tt.want {
				t.Errorf("AddZwsp() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkAddZwsp(b *testing.B) {
	for _, tt := range addZwspTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				AddZwsp(tt.args)
			}
		})
	}
}
