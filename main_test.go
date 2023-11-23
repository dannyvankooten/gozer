package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseFrontMatter(t *testing.T) {
	p := &Page{
		Filepath: "example/content/index.md",
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
		Filepath: "example/content/index.md",
	}

	content, err := p.ParseContent()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(content, "<p>Hey, welcome on my site!</p>") {
		t.Errorf("Invalid content. Got %v", content)
	}
}

func TestParseDateFromFilename(t *testing.T) {
	tests := []struct {
		input string
		want  time.Time
	}{
		{"content/index.md", time.Time{}},
		{"content/2023-11-23-hello-world.md", time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC)},
		{"content/blog/2023-11-23-hello-world.md", time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		got := parseDateFromFilename(tc.input)
		if got != tc.want {
			t.Errorf("expected %v, got %v", tc.want, got)
		}
	}
}

func TestFilepathToUrlpath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"content/index.md", "/"},
		{"content/about.md", "about/"},
		{"content/projects/gozer.md", "projects/gozer/"},
	}

	for _, tc := range tests {
		got := filepathToUrlpath(tc.input)
		if got != tc.want {
			t.Errorf("expected %v, got %v", tc.want, got)
		}
	}
}
