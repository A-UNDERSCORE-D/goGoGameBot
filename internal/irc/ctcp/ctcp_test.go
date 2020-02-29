package ctcp

import (
	"fmt"
	"testing"
)

var ctcpTests = map[string]bool{
	"\x01ACTION test message\x01":                          true,
	"\x01THIS is a malformed CTCP message, but still good": true,
	"\x01TEST\x01":                        true,
	"\x01TEST \x01":                       true,
	"\x01TEST ":                           true,
	"this is not a CTCP message":          false,
	"this is an invalid CTCP message\x01": false,
	"\x01":                                false,
}

func TestIsCTCP(t *testing.T) {
	makeTest := func(str string, isCtcp bool) func(t *testing.T) {
		return func(t *testing.T) {
			if IsCTCP(str) != isCtcp {
				t.Errorf("string %q expected to be %t, was returned as %t", str, isCtcp, !isCtcp)
			}
		}
	}

	for str, isCtcp := range ctcpTests {
		t.Run(fmt.Sprintf("%q", str), makeTest(str, isCtcp))
	}
}

func BenchmarkIsCTCP(b *testing.B) {
	bench := func(str string) func(b *testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = IsCTCP(str)
			}
		}
	}

	for str := range ctcpTests {
		b.Run(str, bench(str))
	}
}

var parsedCtcpTests = map[string]*CTCP{
	"\x01ACTION test message\x01":                          {"ACTION", "test message"},
	"\x01THIS is a malformed CTCP message, but still good": {"THIS", "is a malformed CTCP message, but still good"},
	"\x01TEST\x01":                        {"TEST", ""},
	"\x01TEST \x01":                       {"TEST", ""},
	"\x01TEST ":                           {"TEST", ""},
	"this is not a CTCP message":          nil,
	"this is an invalid CTCP message\x01": nil,
	"\x01":                                nil,
}

func TestParse(t *testing.T) { //nolint:gocognit // well you can either have this, or weird var shit.
	test := func(name string, ctcp *CTCP) func(t *testing.T) {
		return func(t *testing.T) {
			if parsed, err := Parse(name); err == nil {
				if ctcp == nil || parsed.Command != ctcp.Command || parsed.Arg != ctcp.Arg {
					t.Errorf("Incorrectly parsed CTCP string: got %v, expected %v", parsed, parsed)
				}
			} else {
				if ctcp != nil {
					t.Errorf("Incorrectly parsed CTCP string %q: got nil, expected %v", name, ctcp)
				}
			}
		}
	}

	for str, v := range parsedCtcpTests {
		t.Run(fmt.Sprintf("%q", str), test(str, v))
	}
}

func BenchmarkParse(b *testing.B) {
	bench := func(toParse string) func(b *testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = Parse(toParse)
			}
		}
	}

	for str := range parsedCtcpTests {
		b.Run(str, bench(str))
	}
}
