package util

import (
    "text/template"
)

var TemplateUtilFuncs = template.FuncMap{
    "zwsp":    AddZwsp,
    "wordEOL": WordEol,
    "escape":  EscapeString,
}
