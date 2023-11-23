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

func TestFilepathToUrlpath(t *testing.T) {
	tests := []struct {
		input                 string
		expectedUrlPath       string
		expectedDatePublished time.Time
	}{
		{input: "content/index.md", expectedUrlPath: "/", expectedDatePublished: time.Time{}},
		{input: "content/about.md", expectedUrlPath: "about/", expectedDatePublished: time.Time{}},
		{input: "content/projects/gozer.md", expectedUrlPath: "projects/gozer/", expectedDatePublished: time.Time{}},
		{input: "content/2023-11-23-hello-world.md", expectedUrlPath: "hello-world/", expectedDatePublished: time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC)},
		{input: "content/blog/2023-11-23-here-we-are.md", expectedUrlPath: "blog/here-we-are/", expectedDatePublished: time.Date(2023, 11, 23, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		urlPath, datePublished := parseFilename(tc.input, "")
		if urlPath != tc.expectedUrlPath {
			t.Errorf("expected %v, got %v", tc.expectedUrlPath, urlPath)
		}

		if !datePublished.Equal(tc.expectedDatePublished) {
			t.Errorf("expected %v, got %v", tc.expectedDatePublished, datePublished)
		}
	}
}
