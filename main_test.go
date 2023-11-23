package main

import (
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	p := &Page{
		Path: "example/content/index.md",
	}

	if err := parseFrontMatter(p); err != nil {
		t.Fatal(err)
	}

	expectedTitle := "My site"
	if p.Title != expectedTitle {
		t.Errorf("Invalid title. Expected %v, got %v", expectedTitle, p.Title)
	}
}

func TestParseContent(t *testing.T) {
	p := &Page{
		Path: "example/content/index.md",
	}

	content, err := p.ParseContent()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "<p>Hey, welcome on my site!</p>") {
		t.Errorf("Invalid content. Got %v", content)
	}

}
