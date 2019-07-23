package util

import (
	"strings"

	"github.com/goshuirc/irc-go/ircfmt"
)

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

const zwsp = '\u200b'

// StripAll strips both IRC control codes and any extra weird ascii control codes
func StripAll(s string) string {
	s = ircfmt.Strip(s)
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
