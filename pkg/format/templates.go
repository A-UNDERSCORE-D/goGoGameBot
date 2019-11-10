package format

import (
	"bytes"
	"errors"
	"text/template"

	"git.ferricyanide.solutions/A_D/goGoGameBot/pkg/util"
)

// UtilFuncs is a standard set of functions added to all Format's template.Template
var UtilFuncs = template.FuncMap{
	"zwsp":     util.AddZwsp,
	"wordEOL":  util.WordEol,
	"escape":   util.EscapeString,
	"stripAll": util.StripAll,
	"eat":      func(...interface{}) string { return "" },
}

// Format represents a wrapped template.Template
type Format struct {
	FormatString   string             `xml:",chardata"` // The original format string
	CompiledFormat *template.Template `xml:"-"`         // Our internal template
	compiled       bool               // Have we been compiled yet
}

var (
	// ErrEmptyFormat Is returned when the format is empty
	ErrEmptyFormat = errors.New("format: cannot compile an empty format")
)

// Compile compiles the given format string into a text.template, evaluating IRC colours if requested, and adding the
// default functions plus any passed to the template. if the template is invalid or the format has already been compiled,
// Compile errors. An optional root text/template can be passed, and if so, the compiled format's internal template will
// be associated with the passed root
func (f *Format) Compile(name string, root *template.Template, funcMaps ...template.FuncMap) error {
	if f.compiled {
		return errors.New("format: cannot compile a format twice")
	}

	if f.FormatString == "" {
		return ErrEmptyFormat
	}
	var toSet *template.Template
	if root == nil {
		toSet = template.New(name)
	} else {
		toSet = root.New(name)
	}
	toSet.Funcs(UtilFuncs)
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

// ExecuteBytes is like Execute but returns a slice of bytes
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

// Execute runs a compiled Format and returns the resulting string
func (f *Format) Execute(data interface{}) (string, error) {
	b, err := f.ExecuteBytes(data)
	return string(b), err
}
