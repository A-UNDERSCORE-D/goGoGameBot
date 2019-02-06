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
    FormatString        string             `xml:",chardata"`
    StripNewlines       bool               `xml:"strip_newlines,attr"`
    StripWhitespace     bool               `xml:"strip_whitespace,attr"`
    CompiledFormat      *template.Template `xml:"-"`
    compiled            bool
    whitespaceToReplace string
}

var newlinesReplacer = strings.NewReplacer("\n", "", "\r", "")

func (f *Format) doFmtStringCleanup() {
    if f.StripWhitespace {
        var toSet []string
        for _, l := range strings.Split(f.FormatString, "\n") {
            toSet = append(toSet, strings.Trim(l, " "))
        }
        f.FormatString = strings.Join(toSet, "\n")
    }

    if f.StripNewlines {
        f.FormatString = newlinesReplacer.Replace(f.FormatString)
    }
}

func (f *Format) Compile(name string, funcMaps ...template.FuncMap) error {
    f.doFmtStringCleanup()
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
    return nil
}

func (f *Format) Execute(data interface{}) (string, error) {
    if !f.compiled {
        return "", errors.New("util.Format: cannot execute an uncompiled Format")
    }
    buf := new(bytes.Buffer)
    err := f.CompiledFormat.Execute(buf, data)
    if err != nil {
        return "", err
    }
    return buf.String(), nil
}
