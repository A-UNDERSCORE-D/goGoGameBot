package ctcp

import (
	"errors"
	"strings"
)

const ctcpChar = 0x01
const ctcpCharString = string(ctcpChar)

func IsCTCP(s string) bool {
	return len(s) > 1 && s[0] == ctcpChar
}

type CTCP struct {
	Command string
	Arg     string
}

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
