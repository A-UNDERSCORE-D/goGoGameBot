package util

import (
    "bytes"
    "errors"
    "strings"
    "text/template"

    "github.com/goshuirc/irc-go/ircfmt"
)

var TemplateUtilFuncs = template.FuncMap{
    "zwsp":    AddZwsp,
    "wordEOL": WordEol,
    "escape":  EscapeString,
    "stripColour": ircfmt.Strip,
}

type Format struct {
    FormatString        string             `xml:",chardata"`
    StripNewlines       bool               `xml:"strip_newlines,attr"`
    StripWhitespace     bool               `xml:"strip_whitespace,attr"`
    CompiledFormat      *template.Template `xml:"-"`
    compiled            bool
}

var newlinesReplacer = strings.NewReplacer("\n", "", "\r", "", "\t", "")

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

var (
    ErrEmptyFormat = errors.New("format: cannot compile an empty format")
)

func (f *Format) Compile(name string, evalColour bool, funcMaps ...template.FuncMap) error {
    if f.compiled {
        return errors.New("format: cannot compile a format twice")
    }
    f.doFmtStringCleanup()
    if f.FormatString == "" {
        return ErrEmptyFormat
    }
    toSet := template.New(name)
    toSet.Funcs(TemplateUtilFuncs)
    for _, entry := range funcMaps {
        toSet.Funcs(entry)
    }

    if evalColour {
        f.FormatString = ircfmt.Unescape(f.FormatString)
    }

    res, err := toSet.Parse(f.FormatString)
    if err != nil {
        return err
    }
    f.CompiledFormat = res
    f.compiled = true
    return nil
}

func (f *Format) ExecuteBytes(data interface{}) ([]byte, error) {
    if !f.compiled {
        return nil, errors.New("util.Format: cannot execute an uncompiled Format")
    }
    buf := new(bytes.Buffer)
    err := f.CompiledFormat.Execute(buf, data)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

func (f *Format) Execute(data interface{}) (string, error) {
    b, err := f.ExecuteBytes(data)
    return string(b), err
}
