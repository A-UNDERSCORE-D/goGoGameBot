package util

import (
    "testing"

    "github.com/goshuirc/irc-go/ircmsg"
)

var expectedLine = ircmsg.IrcMessage{Command: "TEST", Params: []string{"test", "args", "are", "very testy"}}

func TestMakeSimpleIRCLine(t *testing.T) {
    res := MakeSimpleIRCLine("TEST", "test", "args", "are", "very testy")
    if res.Command != expectedLine.Command || !strSliceEq(res.Params, expectedLine.Params) {
        t.Errorf("returned line %#v is not the expected %#v", res, expectedLine)
    }
}

func strSliceEq(s1, s2 []string) bool {
    if len(s1) != len(s2) {
        return false
    }
    for i, v := range s1 {
        if v != s2[i] {
            return false
        }
    }
    return true
}
