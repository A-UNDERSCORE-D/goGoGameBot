package util

import "strings"

func CleanSplitOnSpace(s string) []string {
    split := strings.Split(s, " ")
    for i, v := range split {
        if strings.Count(v, " ") == len(v) && len(split) > i{
            split = append(split[:i], split[i+1:]...)
        }
    }
    return split
}

func WordEol(s string, wordIdx int) string {
    var eol []string
    splitMsg := strings.Split(s, " ")
    for i := range splitMsg {
        eol = append(eol, strings.Join(splitMsg[i:], " "))
    }
    if wordIdx < len(eol){
        //noinspection ALL
        return eol[wordIdx]
    }
    return ""
}

var escapeReplacer = strings.NewReplacer(`\`, `\\`, `'`, `\'`, `"`, `\"`)
func EscapeString(s string) string {
    return escapeReplacer.Replace(s)
}
