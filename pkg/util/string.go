package util

import (
	"fmt"
	"strings"
)

// CleanSplitOnSpace splits the given string on space specifically without adding empty strings to the resulting array
// for repeated spaces
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

var escapeReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`, "\n", `:`, "\r", ``)

// EscapeString escapes commonly (ab)used strings
func EscapeString(s string) string {
	return escapeReplacer.Replace(s)
}

const zwsp = '\u200b'

// StripAll strips any extra weird ascii control codes
func StripAll(s string) string {
	return strings.Map(func(r rune) rune {
		if r < ' ' || r == zwsp {
			return -1
		}
		return r
	}, s)
}

// IdxOrEmpty returns the given slice index, or an empty string
func IdxOrEmpty(slice []string, idx int) string {
	if len(slice) > idx {
		return slice[idx]
	}
	return ""
}

// JoinToMaxLength takes a string slice and joins it on sep until the joined string cannot be made larger without
// crossing maxLength length. If any entry in the slice to be joined is larger than maxLength, it will be added on its
// own to an entry in the resulting slice
func JoinToMaxLength(toJoin []string, sep string, maxLength int) []string {
	var out []string
	curBuilder := strings.Builder{}
	curBuilder.Grow(maxLength)
	for _, s := range toJoin {
		entryLen := len(s)
		if curBuilder.Len() == 0 && entryLen > maxLength {
			out = append(out, s)
			continue
		}

		if curBuilder.Len() > 0 && curBuilder.Len()+len(sep)+entryLen > maxLength {
			out = append(out, curBuilder.String())
			curBuilder.Reset()
		}
		if curBuilder.Len() != 0 {
			curBuilder.WriteString(sep)
		}
		curBuilder.WriteString(s)

	}
	if curBuilder.Len() > 0 {
		out = append(out, curBuilder.String())
	}
	return out
}

func abs(n int) int {
	if n >= 0 {
		return n
	}
	return -n
}

// ReverseIdx returns either the given index, or, if the given index is negative, that index starting from the end
// of the slice. Much like a python lists behaviour when indexed with a negative number
func ReverseIdx(toIdx []string, idx int) string {
	if idx >= 0 {
		return IdxOrEmpty(toIdx, idx)
	}

	if abs(idx) > len(toIdx) {
		return ""
	}

	return IdxOrEmpty(toIdx, len(toIdx)+idx)
}

// AddZwsp adds a zero width space to the given string if its length is greater than two
func AddZwsp(s string) string {
	if len(s) < 2 {
		return s
	}

	return fmt.Sprintf("%c\u200b%s", s[0], s[1:])
}
