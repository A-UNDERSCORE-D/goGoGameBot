package util

import (
    "fmt"
    "strings"
)

func MakeColourMap(mapIn map[string]string) (ret *strings.Replacer, err error) {
    zipped := getOldNewSlice(mapIn)
    defer func() {
        if res := recover(); res != nil {
            ret = nil
            err = fmt.Errorf("could not create strings.Replacer: %v", res)
        }
    }()
    return strings.NewReplacer(zipped...), nil
}

func getOldNewSlice(mapIn map[string]string) []string {
    zipped := []string{"$$", "$"} // Start out with the escaped $ that ircfmt supports
    for k, v := range mapIn {
        var toAppendOld string
        switch k {
        case "bold":
            toAppendOld = "$b"
        case "italic":
            toAppendOld = "$i"
        case "reverse_colour":
            toAppendOld = "$v"
        case "strikethrough":
            toAppendOld = "$s"
        case "underline":
            toAppendOld = "$u"
        case "monospace":
            toAppendOld = "$m"
        case "reset":
            toAppendOld = "$r"
        case "white":
            toAppendOld = "$c[white]"
        case "black":
            toAppendOld = "$c[black]"
        case "blue":
            toAppendOld = "$c[blue]"
        case "green":
            toAppendOld = "$c[green]"
        case "red":
            toAppendOld = "$c[red]"
        case "brown":
            toAppendOld = "$c[brown]"
        case "magenta":
            toAppendOld = "$c[magenta]"
        case "orange":
            toAppendOld = "$c[orange]"
        case "yellow":
            toAppendOld = "$c[yellow]"
        case "light_green":
            toAppendOld = "$c[light green]"
        case "cyan":
            toAppendOld = "$c[cyan]"
        case "light_cyan":
            toAppendOld = "$c[light cyan]"
        case "light_blue":
            toAppendOld = "$c[light blue]"
        case "pink":
            toAppendOld = "$c[pink]"
        case "grey":
            toAppendOld = "$c[grey]"
        case "light_grey":
            toAppendOld = "$c[light grey]"
        case "default":
            toAppendOld = "$c[default]"
        }
        if toAppendOld != "" {
            zipped = append(zipped, toAppendOld, v)
        }
    }
    return zipped
}
