package util

import (
    "bytes"
    "errors"
    "strings"
    "text/template"
)

var TemplateUtilFuncs = template.FuncMap{
    "zwsp":    AddZwsp,
    "wordEOL": WordEol,
    "escape":  EscapeString,
}

type Format struct {
    FormatString        string             `xml:",innerxml"`
    StripNewlines       bool               `xml:"strip_newlines,attr"`
    StripWhitespace     int                `xml:"strip_whitespace,attr"`
    CompiledFormat      *template.Template `xml:"-"`
    compiled            bool
    whitespaceToReplace string
}

func (f *Format) Compile(name string, funcMaps ...template.FuncMap) error {
    toSet := template.New(name)
    toSet.Funcs(TemplateUtilFuncs)
    for _, entry := range funcMaps {
        toSet.Funcs(entry)
    }
    res, err := toSet.Parse(f.FormatString)
    if err != nil {
        return err
    }
    f.CompiledFormat = res
    f.compiled = true
    f.whitespaceToReplace = strings.Repeat(" ", f.StripWhitespace)
    return nil
}

var newlinesReplacer = strings.NewReplacer("\n", "", "\r", "")

func (f *Format) Execute(data interface{}) (string, error) {
    if !f.compiled {
        return "", errors.New("util.Format: cannot execute an uncompiled Format")
    }
    buf := new(bytes.Buffer)
    err := f.CompiledFormat.Execute(buf, data)
    if err != nil {
        return "", err
    }
    out := buf.String()
    if f.StripNewlines {
        out = newlinesReplacer.Replace(out)
    }
    if f.StripWhitespace != 0 {
        out = strings.Replace(out, f.whitespaceToReplace, "", -1)
    }
    return out, nil
}
