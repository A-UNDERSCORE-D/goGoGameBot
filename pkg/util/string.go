package util

import "strings"

// CleanSplitOnSpace splits the given string on space specifically without adding empty strings to the resulting array for
// repeated spaces
func CleanSplitOnSpace(s string) []string {
	split := strings.Split(s, " ")
	var out []string
	for _, v := range split {
		if len(v) == 0 {
			continue
		}
		out = append(out, v)
	}
	return out
}

func WordEol(s string, wordIdx int) string {
	var eol []string
	splitMsg := strings.Split(s, " ")
	for i := range splitMsg {
		eol = append(eol, strings.Join(splitMsg[i:], " "))
	}
	if wordIdx < len(eol) {
		return eol[wordIdx]
	}
	return ""
}

var escapeReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`)

func EscapeString(s string) string {
	return escapeReplacer.Replace(s)
}
