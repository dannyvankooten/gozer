package main

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
	goldmark.WithExtensions(extension.GFM, extension.Footnote))

var frontMatter = []byte("+++")
