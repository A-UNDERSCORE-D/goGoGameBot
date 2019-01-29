package util

import "strings"

func CleanSplitOnSpace(s string) []string {
    split := strings.Split(s, " ")
    for i, v := range split {
        if strings.Count(v, " ") == len(v) {
            split = append(split[:i], split[i+1:]...)
        }
    }
    return split
}
