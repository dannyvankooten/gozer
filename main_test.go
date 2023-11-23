package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExampleSite(t *testing.T) {
	_ = os.RemoveAll("build/")
	buildSite("example/", "config.xml")

	tests := []struct {
		file     string
		contains []byte
	}{
		{"index.html", []byte("<p>Hey, welcome on my site!</p>")},
		{"about/index.html", []byte("<li>Dolor</li>")},
		{"hello-world/index.html", []byte("This is a blog post.")},
		{"favicon.ico", []byte{}},
		{"feed.xml", []byte{}},
		{"sitemap.xml", []byte("<url><loc>http://localhost:8080//</loc>")},
		{"sitemap.xsl", []byte{}},
	}

	for _, tc := range tests {
		content, err := os.ReadFile("build/" + tc.file)
		if err != nil {
			t.Errorf("Expected file, got error: %s", err)
		}

		if !bytes.Contains(content, tc.contains) {
			t.Errorf("Output file %s does not have expected content %s", tc.file, tc.contains)
		}
	}
}

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
