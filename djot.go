package main

import (
	"github.com/sivukhin/godjot/v2/djot_html"
	"github.com/sivukhin/godjot/v2/djot_parser"
)

func ConvertDjot(content []byte) string {
	ast := djot_parser.BuildDjotAst(content)
	return djot_html.New().ConvertDjot(&djot_html.HtmlWriter{}, ast...).String()
}
