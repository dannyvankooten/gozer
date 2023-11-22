package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

var templates = template.Must(template.ParseFS(os.DirFS("templates/"), "*.html"))

type Config struct {
	SiteUrl string `xml:"site_url"`
}

type Page struct {
	Title      string
	Template   string
	Date       time.Time
	Path       string
	PrettyName string
	IsBlogPost bool
	LastMod    time.Time
}

func (p *Page) URLPath() string {
	path := strings.TrimSuffix(strings.TrimSuffix(p.Dest(), "/index.html"), "/")
	if len(path) > 0 {
		if path[0] == '/' {
			path = path[1:]
		}

		if path[len(path)-1] != '/' {
			path += "/"
		}
	}
	return path
}

func (p *Page) Dest() string {
	d := strings.TrimPrefix(p.Path, "content")

	if p.PrettyName != "" {
		d = strings.TrimSuffix(d, filepath.Base(d))
		d += p.PrettyName
	}

	d = strings.TrimSuffix(d, ".md")
	d = strings.TrimSuffix(d, ".html")
	d = strings.TrimSuffix(d, "_index")

	d += "/index.html"
	return d
}

func parseFrontMatter(p *Page) error {
	// open file to read front matter
	fh, err := os.Open(p.Path)
	if err != nil {
		return err
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "+++" {
			break
		}

		pos := strings.Index(line, "=")
		if pos == -1 {
			continue
		}

		name := strings.TrimSpace(line[0:pos])
		value := strings.TrimSpace(line[pos+1:])
		if value[0] == '"' {
			value = strings.Trim(value, "\"")
		}

		switch name {
		case "title":
			p.Title = value
		case "template":
			p.Template = value
		case "date":
			// discard, we get this from filename only now
		default:
			log.Warn("Unknown front-matter key in %s: %#v\n", p.Path, name)
		}

	}

	return nil
}

func (p *Page) ParseContent() (string, error) {
	fileContent, err := os.ReadFile(p.Path)
	if err != nil {
		return "", err
	}

	if len(fileContent) < 6 {
		return "", errors.New("missing front matter")
	}

	// Skip front matter
	pos := bytes.Index(fileContent[3:], []byte("+++"))
	if pos > -1 {
		fileContent = fileContent[pos+6:]
	}

	// TODO: Only process Markdown if this is a .md file
	var buf bytes.Buffer
	if err := md.Convert(fileContent, &buf); err != nil {
		return "", err
	}

	return string(buf.Bytes()), nil
}

func (s *Site) buildPage(p *Page) error {
	content, err := p.ParseContent()
	if err != nil {
		return err
	}

	if err := os.MkdirAll("build/"+filepath.Dir(p.Dest()), 0755); err != nil {
		return err
	}

	fh, err := os.Create("build/" + p.Dest())
	if err != nil {
		return err
	}
	defer fh.Close()

	tmpl := templates.Lookup(p.Template)
	if tmpl == nil {
		return errors.New(fmt.Sprintf("Invalid template name: %s", p.Template))
	}

	return tmpl.Execute(fh, map[string]any{
		"Page":    p,
		"Posts":   s.posts,
		"Pages":   s.pages,
		"SiteUrl": s.config.SiteUrl,
		"Title":   p.Title,
		"Content": template.HTML(content),
	})
}

func copyFile(src string, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// if it's a dir, just re-create it in build/
	if info.IsDir() {
		err := os.Mkdir(dest, info.Mode())
		if err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}

		return nil
	}

	// open input
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// create output
	fh, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fh.Close()

	// match file permissions
	err = fh.Chmod(info.Mode())
	if err != nil {
		return err
	}

	// copy content
	_, err = io.Copy(fh, in)
	return err
}

func copyDirRecursively(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		outpath := dst + strings.TrimPrefix(path, src)
		return copyFile(path, outpath)
	})
}

type Site struct {
	pages  []Page
	posts  []Page
	config *Config
}

func (s *Site) readContent() error {

	err := filepath.WalkDir("content", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		p := Page{
			Path:       path,
			IsBlogPost: strings.HasPrefix(path, "content/blog/") && !strings.HasSuffix(path, "index.md"),
			LastMod:    info.ModTime(),
			Template:   "default.html",
		}

		// parse date from filename
		filename := filepath.Base(p.Path)
		if len(filename) > 11 && filename[4] == '-' && filename[7] == '-' {
			date, err := time.Parse("2006-01-02", filename[0:len("2000-01-02")])
			if err == nil {
				p.Date = date
				p.PrettyName = filename[11:]
			}
		}

		if err := parseFrontMatter(&p); err != nil {
			return err
		}

		s.pages = append(s.pages, p)

		if p.IsBlogPost {
			s.posts = append(s.posts, p)
		}
		return nil
	})

	// sort posts by date
	sort.Slice(s.posts, func(i int, j int) bool {
		return s.posts[i].Date.After(s.posts[j].Date)
	})

	return err
}

func (s *Site) createSitemap() error {
	type Url struct {
		XMLName xml.Name `xml:"url"`
		Loc     string   `xml:"loc"`
		LastMod string   `xml:"lastmod"`
	}

	type Envelope struct {
		XMLName        xml.Name `xml:"urlset"`
		XMLNS          string   `xml:"xmlns,attr"`
		SchemaLocation string   `xml:"xsi:schemaLocation,attr"`
		XSI            string   `xml:"xmlns:xsi,attr"`
		Image          string   `xml:"xmlns:image,attr"`
		Urls           []Url    `xml:""`
	}

	urls := make([]Url, 0, len(s.pages))
	for _, p := range s.pages {
		urls = append(urls, Url{
			Loc:     s.config.SiteUrl + p.URLPath(),
			LastMod: p.LastMod.Format(time.RFC1123Z),
		})
	}

	env := Envelope{
		SchemaLocation: "http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd http://www.google.com/schemas/sitemap-image/1.1 http://www.google.com/schemas/sitemap-image/1.1/sitemap-image.xsd",
		XMLNS:          "http://www.sitemaps.org/schemas/sitemap/0.9",
		XSI:            "http://www.w3.org/2001/XMLSchema-instance",
		Image:          "http://www.google.com/schemas/sitemap-image/1.1",
		Urls:           urls,
	}

	wr, err := os.Create("build/sitemap.xml")
	if err != nil {
		return err
	}
	defer wr.Close()

	enc := xml.NewEncoder(wr)
	if _, err := wr.WriteString(`<?xml version="1.0" encoding="UTF-8"?><?xml-stylesheet type="text/xsl" href="/sitemap.xsl"?>` + "\n"); err != nil {
		return err
	}
	if err := enc.Encode(env); err != nil {
		return err
	}

	// copy xml stylesheet
	return copyFile("sitemap.xsl", "build/sitemap.xsl")
}

func (s *Site) createRSSFeed() error {

	type Item struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"`
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
		GUID        string `xml:"guid"`
	}

	type Channel struct {
		Title         string `xml:"title"`
		Link          string `xml:"link"`
		Description   string `xml:"description"`
		Generator     string `xml:"generator"`
		LastBuildDate string `xml:"lastBuildDate"`
		Items         []Item `xml:"item"`
	}

	type Feed struct {
		XMLName xml.Name `xml:"rss"`
		Version string   `xml:"version,attr"`
		Atom    string   `xml:"xmlns:atom,attr"`
		Channel Channel  `xml:"channel"`
	}

	// add 10 most recent posts to feed
	n := min(len(s.posts), 10)
	items := make([]Item, 0, n)
	for _, p := range s.posts[0:n] {
		pageContent, err := p.ParseContent()
		if err != nil {
			continue
		}

		items = append(items, Item{
			Title:       p.Title,
			Link:        s.config.SiteUrl + p.URLPath(),
			Description: pageContent,
			PubDate:     p.Date.Format(time.RFC1123Z),
			GUID:        s.config.SiteUrl + p.URLPath(),
		})
	}

	feed := Feed{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:         "Site title",
			Link:          s.config.SiteUrl,
			Generator:     "Gosite",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	wr, err := os.Create("build/feed.xml")
	if err != nil {
		return err
	}
	defer wr.Close()

	enc := xml.NewEncoder(wr)
	if _, err := wr.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"); err != nil {
		return err
	}
	if err := enc.Encode(feed); err != nil {
		return err
	}

	return nil
}

func parseConfig() (*Config, error) {
	wr, err := os.Open("config.xml")
	if err != nil {
		return nil, err
	}
	defer wr.Close()
	var config Config
	if err := xml.NewDecoder(wr).Decode(&config); err != nil {
		return nil, err
	}

	config.SiteUrl = strings.TrimSuffix(config.SiteUrl, "/") + "/"

	return &config, nil
}

func main() {
	var err error
	timeStart := time.Now()

	site := Site{}

	// read config.xml
	site.config, err = parseConfig()
	if err != nil {
		log.Fatal("Error reading config.xml: %w\n", err)
	}

	// read content
	if err := site.readContent(); err != nil {
		log.Fatal("Error reading content/: %s", err)
	}

	// build each individual page
	for _, p := range site.pages {
		if err := site.buildPage(&p); err != nil {
			log.Warn("Error processing %s: %s\n", p.Path, err)
		}
	}

	// create sitemap
	if err := site.createSitemap(); err != nil {
		log.Warn("Error creating sitemap: %s\n", err)
	}

	// create sitemap
	if err := site.createRSSFeed(); err != nil {
		log.Warn("Error creating RSS feed: %s\n", err)
	}

	// static files
	if err := copyDirRecursively("public", "build"); err != nil {
		log.Fatal("Error copying public/ directory: %s", err)
	}

	log.Info("Built site containing %d pages in %d ms\n", len(site.pages), time.Since(timeStart).Milliseconds())

	if len(os.Args) > 1 && os.Args[1] == "serve" {
		log.Info("Listening on http://localhost:8080\n")
		log.Fatal("Hello", http.ListenAndServe("localhost:8080", http.FileServer(http.Dir("build/"))))
	}
}

// TODO:
// - Go through all files and build a list of pages
// - Then start writing output files, passing the list of posts to each page
