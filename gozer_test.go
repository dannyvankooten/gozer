package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestExampleSite(t *testing.T) {
	_ = os.RemoveAll("build/")
	buildSite("example/", "config.toml")

	tests := []struct {
		file     string
		contains [][]byte
	}{
		{"index.html", [][]byte{
			[]byte("<p>Hey, welcome on my site!</p>"),
			[]byte("<title>My site</title>")},
		},
		{"about/index.html", [][]byte{
			[]byte("<title>About me</title>"),
			[]byte("<li>Dolor</li>")},
		},
		{"hello-world/index.html", [][]byte{
			[]byte("<title>Hello, world!</title>"),
			[]byte("This is a blog post.")},
		},
		{"favicon.ico", nil},
		{"feed.xml", [][]byte{
			[]byte("<item><title>Hello, world!</title><link>http://localhost:8080/hello-world/</link>"),
		}},
		{"sitemap.xml", [][]byte{
			[]byte("<url><loc>http://localhost:8080/</loc>"),
		}},
		{"sitemap.xsl", nil},
	}

	for _, tc := range tests {
		content, err := os.ReadFile("build/" + tc.file)
		if err != nil {
			t.Errorf("Expected file, got error: %s", err)
		}

		for _, e := range tc.contains {
			if !bytes.Contains(content, e) {
				t.Errorf("Output file %s does not have expected content %s", tc.file, e)
			}
		}

	}
}

func TestParseConfigFile(t *testing.T) {
	s := Site{}
	if err := parseConfig(&s, "example/config.toml"); err != nil {
		t.Errorf("error parsing config file: %s", err)
	}

	expectedSiteUrl := "http://localhost:8080/"
	expectedTitle := "My website"
	if s.SiteUrl != expectedSiteUrl {
		t.Errorf("invalid site url. expected %v, got %v", expectedSiteUrl, s.SiteUrl)
	}

	if s.Title != expectedTitle {
		t.Errorf("invalid site title. expected %v, got %v", expectedTitle, s.Title)
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

	if content != "<p>Hey, welcome on my site!</p>\n" {
		t.Errorf("Invalid content. Got %v", content)
	}
}

func TestFilepathToUrlpath(t *testing.T) {
	tests := []struct {
		input                 string
		expectedUrlPath       string
		expectedDatePublished time.Time
	}{
		{input: "content/index.md", expectedUrlPath: "", expectedDatePublished: time.Time{}},
		{input: "content/about.md", expectedUrlPath: "about/", expectedDatePublished: time.Time{}},
		{input: "content/blog/index.md", expectedUrlPath: "blog/", expectedDatePublished: time.Time{}},
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

func BenchmarkParseFrontMatter(b *testing.B) {
	data := `+++
title = "My page title"
template = "Page template"
+++

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur ac pretium magna. Phasellus ut ligula vel erat dictum sollicitudin eu a dolor. Donec orci mauris, cursus eget elementum eu, tempor sed massa. Aliquam mattis ullamcorper metus, sodales fermentum lectus fringilla id. Duis dui ligula, lobortis ut leo id, semper ultricies justo. Etiam vehicula sit amet ligula vitae maximus. Aenean consectetur nisl ac est convallis, vel dictum nulla iaculis. Ut dignissim lobortis ipsum, vel molestie lectus ornare quis. In hac habitasse platea dictumst. Sed ut elementum nulla.

Ut eleifend felis lacus, id condimentum purus laoreet et. Nam sodales mi cursus, porta enim aliquet, venenatis quam. Sed et quam nisl. Donec libero ex, eleifend sit amet dui at, fermentum semper sem. Donec gravida id nibh eu mollis. Fusce pellentesque gravida ipsum, sit amet sagittis tellus. Donec consectetur nulla enim. Donec quis ornare tellus. Maecenas eget imperdiet lacus. Ut imperdiet dui nisi, a tristique metus sodales vel.

Proin non ex id erat feugiat imperdiet. Duis posuere finibus quam, quis blandit lorem vehicula sit amet. Fusce pulvinar commodo magna, ut sodales massa interdum at. Nam in dapibus nunc. Vestibulum nisi nisl, vestibulum ac vestibulum in, maximus vitae nibh. Nulla egestas pellentesque velit, vitae tempor massa scelerisque et. Nam nibh metus, vestibulum eu justo vitae, venenatis faucibus eros. Suspendisse sed ligula dolor. Etiam sed elit ullamcorper, placerat ex at, rutrum est. Quisque vitae dolor non metus lobortis sagittis. Donec pretium orci aliquam tortor blandit, sed consectetur massa ullamcorper. Praesent vehicula nunc quis urna tempor finibus. Etiam lectus urna, tempor eget diam ac, consequat commodo nunc. Nullam sed venenatis ante, ac mollis lorem. Nunc vitae faucibus lectus. Vivamus vel arcu justo.

Quisque rhoncus elementum sapien ac semper. Sed tristique elit vel nibh semper tincidunt. Nunc feugiat massa eget magna accumsan, eu commodo tortor accumsan. Morbi porttitor metus nec tellus bibendum, in consectetur ex rhoncus. Phasellus dapibus tincidunt ligula, in posuere ligula. Praesent vestibulum porttitor lorem nec mollis. Praesent magna dui, bibendum sed malesuada a, tristique id ipsum. Phasellus non ipsum eget est vulputate rutrum non nec leo. Nam consequat lobortis lorem non accumsan. Aliquam eget elit in dolor malesuada consequat. Sed luctus bibendum arcu eu posuere. Integer nec ipsum turpis. Sed luctus risus ante, eget gravida magna auctor eget sodales sed.
`

	filepath := os.TempDir() + "/front-matter.md"
	err := os.WriteFile(filepath, []byte(data), 0655)
	if err != nil {
		return
	}

	p := &Page{
		Filepath: filepath,
	}
	for n := 0; n < b.N; n++ {
		if err := parseFrontMatter(p); err != nil {
			b.Error(err)
		}
	}
}
