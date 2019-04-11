package util

import (
    "fmt"
    "strings"
)

func MakeColourMap(mapIn map[string]string) (ret *strings.Replacer, err error) {
    defer func() {
        if res := recover(); res != nil {
            ret = nil
            err = fmt.Errorf("could not create strings.Replacer: %v", res)
        }
    }()
    return strings.NewReplacer(zip(mapIn)...), nil
}

func zip(mapIn map[string]string) []string {
    var out []string
    for k, v := range mapIn {
        out = append(out, k, v)
    }
    return out
}
