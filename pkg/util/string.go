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

// WordEol returns the given string with wordIdx words (space separations) removed
func WordEol(s string, wordIdx int) string {
	split := strings.Split(s, " ")
	if wordIdx > -1 && len(split) >= wordIdx {
		return strings.Join(split[wordIdx:], " ")
	}
	return ""
}

var escapeReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`)

// EscapeString escapes commonly (ab)used strings
func EscapeString(s string) string {
	return escapeReplacer.Replace(s)
}
