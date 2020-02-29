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
	for str, isCtcp := range ctcpTests {
		t.Run(fmt.Sprintf("%q", str), func(t *testing.T) {
			if IsCTCP(str) != isCtcp {
				t.Errorf("string %q expected to be %t, was returned as %t", str, isCtcp, !isCtcp)
			}
	}
}

func BenchmarkIsCTCP(b *testing.B) {
	for str := range ctcpTests {
		b.Run(str, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = IsCTCP(str)
			}
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

func TestParse(t *testing.T) {
	for str, v := range parsedCtcpTests {
		t.Run(fmt.Sprintf("%q", str), func(t *testing.T) {
			if parsed, err := Parse(str); err == nil {
				if v == nil || parsed.Command != v.Command || parsed.Arg != v.Arg {
					t.Errorf("Incorrectly parsed CTCP string: got %v, expected %v", parsed, parsed)
				}
			} else {
				if v != nil {
					t.Errorf("Incorrectly parsed CTCP string %q: got nil, expected %v", str, v)
				}
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	for str := range parsedCtcpTests {
		b.Run(str, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = Parse(str)
			}
		})
	}
}
