package ctcp

import (
	"errors"
	"strings"
)

const ctcpChar = 0x01
const ctcpCharString = string(ctcpChar)

// IsCTCP returns whether or not the given string is a valid CTCP command
func IsCTCP(s string) bool {
	return len(s) > 1 && s[0] == ctcpChar
}

// CTCP represents a CTCP command and argument
type CTCP struct {
	Command string
	Arg     string
}

// Parse takes a string and returns a CTCP struct representation of the command. If the passed string is not a valid
// CTCP string, Parse returns an error
func Parse(s string) (CTCP, error) {
	if !IsCTCP(s) {
		return CTCP{}, errors.New("not a CTCP string")
	}

	splitMsg := strings.SplitN(s, " ", 2)
	cmd := splitMsg[0]
	args := ""

	if len(splitMsg) > 1 {
		args = splitMsg[1]
	}

	return CTCP{strings.ToUpper(strings.Trim(cmd, ctcpCharString)), strings.Trim(args, ctcpCharString)}, nil
}
