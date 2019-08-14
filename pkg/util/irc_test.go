package util

import (
	"encoding/base64"
	"reflect"
	"regexp"
	"testing"

	"github.com/goshuirc/irc-go/ircmsg"
)

type simpleIRCLineArgs struct {
	command string
	params  []string
}

var simpleIRCLineTests = []struct {
	name string
	args simpleIRCLineArgs
	want ircmsg.IrcMessage
}{
	{
		"simple message",
		simpleIRCLineArgs{
			"TEST",
			[]string{"test", "wordEolArgs", "are", "very testy"},
		},
		ircmsg.IrcMessage{Command: "TEST", Params: []string{"test", "wordEolArgs", "are", "very testy"}},
	},
	{
		"no simpleIRCLineArgs",
		simpleIRCLineArgs{
			"TEST",
			[]string(nil),
		},
		ircmsg.IrcMessage{Command: "TEST", Params: []string(nil)},
	},
	{
		"one arg",
		simpleIRCLineArgs{
			"TEST",
			[]string{"test"},
		},
		ircmsg.IrcMessage{Command: "TEST", Params: []string{"test"}},
	},
}

func TestMakeSimpleIRCLine(t *testing.T) {
	for _, tt := range simpleIRCLineTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeSimpleIRCLine(tt.args.command, tt.args.params...); !ircMsgEq(got, tt.want) {
				t.Errorf("MakeSimpleIRCLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkMakeSimpleIRCLine(b *testing.B) {
	for _, tt := range simpleIRCLineTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				MakeSimpleIRCLine(tt.args.command, tt.args.params...)
			}
		})
	}
}

func ircMsgEq(a, b ircmsg.IrcMessage) bool {
	if a.Command != b.Command {
		return false
	}

	if a.Prefix != b.Prefix {
		return false
	}

	if !reflect.DeepEqual(a.AllTags(), b.AllTags()) {
		return false
	}

	if len(a.Params) != len(b.Params) {
		return false
	}

	for i, p := range a.Params {
		if b.Params[i] != p {
			return false
		}
	}
	// We're out of tests, it must be equal at this point
	return true
}

func TestGenerateSASLString(t *testing.T) {
	type args struct {
		nick         string
		saslUsername string
		saslPasswd   string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"standard",
			args{
				"test",
				"testacct",
				"testpasswd",
			},
			"dGVzdAB0ZXN0YWNjdAB0ZXN0cGFzc3dk",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateSASLString(tt.args.nick, tt.args.saslUsername, tt.args.saslPasswd); got != tt.want {
				res1, _ := base64.StdEncoding.DecodeString(got)
				res2, _ := base64.StdEncoding.DecodeString(tt.want)
				t.Errorf("GenerateSASLString() = %q == %q, want %q == %q", got, res1, tt.want, res2)
			}
		})
	}
}

func BenchmarkGenerateSASLString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateSASLString("test", "testnick", "testpasswd")
	}
}

var globToRegexpTests = []struct {
	name string
	mask string
	want string
}{
	{
		"wildcard nuh",
		"*!*@*",
		".*!.*@.*",
	},
	{
		"empty string",
		"",
		"",
	},
	{
		"glob single character",
		"*!?id*@*",
		".*!.id.*@.*",
	},
	{
		"escaping",
		"..!..@.",
		`\.\.!\.\.@\.`,
	},
	{
		"no special chars",
		"special chars are down the hall on the left",
		"special chars are down the hall on the left",
	},
}

func TestGlobToRegexp(t *testing.T) {
	for _, tt := range globToRegexpTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GlobToRegexp(tt.mask); got.String() != tt.want {
				t.Errorf("GlobToRegexp() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkGlobToRegexp(b *testing.B) {
	for _, tt := range globToRegexpTests {
		regexpCache = make(map[string]*regexp.Regexp)
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GlobToRegexp(tt.mask)
			}
		})
	}
}

type maskMatchArgs struct {
	toCheck string
	masks   []string
}

var maskMatchTests = []struct {
	name string
	args maskMatchArgs
	want bool
}{
	{
		"no masks",
		maskMatchArgs{
			"test",
			nil,
		},
		false,
	},
	{
		"good mask",
		maskMatchArgs{
			"someone!someidentity@somewhere",
			[]string{"someone!*@*"},
		},
		true,
	},
	{
		"bad mask",
		maskMatchArgs{
			"someone@42",
			[]string{"*!*@47"},
		},
		false,
	},
	{
		"second matches",
		maskMatchArgs{
			"test!test@test",
			[]string{"test", "test!*"},
		},
		true,
	},
}

func TestAnyMaskMatch(t *testing.T) {
	for _, tt := range maskMatchTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AnyMaskMatch(tt.args.toCheck, tt.args.masks); got != tt.want {
				t.Errorf("AnyMaskMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkAnyMaskMatch(b *testing.B) {
	for _, tt := range maskMatchTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				AnyMaskMatch(tt.args.toCheck, tt.args.masks)
			}
		})
	}
}
