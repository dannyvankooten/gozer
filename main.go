package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var md = goldmark.New(
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
	),
)

var frontMatter = []byte("+++")

var templates *template.Template

//go:embed sitemap.xsl
var sitemapXSL []byte

type Config struct {
	SiteUrl string `xml:"site_url"`
}

type Page struct {
	Title         string
	Template      string
	DatePublished time.Time
	DateModified  time.Time
	Permalink     string
	UrlPath       string
	Filepath      string
}

// parseFilename parses the URL path and optional date component from the given file path
func parseFilename(path string, rootDir string) (string, time.Time) {
	path = strings.TrimPrefix(path, rootDir+"content/")
	path = strings.TrimSuffix(path, ".md")
	path = strings.TrimSuffix(path, "index")

	filename := filepath.Base(path)
	if len(filename) > 11 && filename[4] == '-' && filename[7] == '-' && filename[10] == '-' {
		date, err := time.Parse("2006-01-02", filename[0:10])
		if err == nil {
			return path[0:len(path)-len(filename)] + filename[11:] + "/", date
		}
	}

	path += "/"
	return path, time.Time{}
}

func parseFrontMatter(p *Page) error {
	// open file to read front matter
	fh, err := os.Open(p.Filepath)
	if err != nil {
		return err
	}
	defer fh.Close()
	scanner := bufio.NewScanner(fh)

	// opening front matter
	// TODO: Make front matter optional?
	scanner.Scan()
	if !bytes.Equal(scanner.Bytes(), frontMatter) {
		return errors.New("missing front matter")
	}

	for scanner.Scan() {
		line := scanner.Text()

		// quit if at end of front matter
		if line == string(frontMatter) {
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
			log.Warn("Unknown front-matter key in %s: %#v\n", p.Filepath, name)
		}

	}

	return nil
}

func (p *Page) ParseContent() (string, error) {
	fileContent, err := os.ReadFile(p.Filepath)
	if err != nil {
		return "", err
	}

	if len(fileContent) < 6 {
		return "", errors.New("missing front matter")
	}

	// Skip front matter
	pos := bytes.Index(fileContent[3:], frontMatter)
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

	dest := p.UrlPath + "index.html"
	if err := os.MkdirAll("build/"+filepath.Dir(dest), 0655); err != nil {
		return err
	}

	fh, err := os.Create("build/" + dest)
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
		"SiteUrl": s.SiteUrl,
		"Title":   p.Title,
		"Content": template.HTML(content),
	})
}

type Site struct {
	pages []Page
	posts []Page

	SiteUrl string
	RootDir string
}

func (s *Site) AddPageFromFile(file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	urlPath, datePublished := parseFilename(file, s.RootDir)

	p := Page{
		Filepath:      file,
		UrlPath:       urlPath,
		DatePublished: datePublished,
		DateModified:  info.ModTime(),
		Template:      "default.html",
	}

	p.Permalink = s.SiteUrl + p.UrlPath

	if err := parseFrontMatter(&p); err != nil {
		return err
	}

	s.pages = append(s.pages, p)

	// every page with a date is assumed to be a blog post
	if !p.DatePublished.IsZero() {
		s.posts = append(s.posts, p)
	}

	return nil
}

func (s *Site) readContent(dir string) error {
	// walk over files in "content" directory
	err := filepath.WalkDir(dir, func(file string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		return s.AddPageFromFile(file)
	})

	// sort posts by date
	sort.Slice(s.posts, func(i int, j int) bool {
		return s.posts[i].DatePublished.After(s.posts[j].DatePublished)
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
			Loc:     s.SiteUrl + p.UrlPath,
			LastMod: p.DateModified.Format(time.RFC1123Z),
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
	if err := os.WriteFile("build/sitemap.xsl", sitemapXSL, 0655); err != nil {
		return err
	}

	return nil
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
			Link:        s.SiteUrl + p.UrlPath,
			Description: pageContent,
			PubDate:     p.DatePublished.Format(time.RFC1123Z),
			GUID:        s.SiteUrl + p.UrlPath,
		})
	}

	feed := Feed{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:         "Site title",
			Link:          s.SiteUrl,
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

func parseConfig(file string) (*Config, error) {
	wr, err := os.Open(file)
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
	configFile := "config.xml"
	rootPath := ""

	// parse flags
	flag.StringVar(&configFile, "config", configFile, "")
	flag.StringVar(&configFile, "c", configFile, "")
	flag.StringVar(&rootPath, "root", rootPath, "")
	flag.StringVar(&rootPath, "r", rootPath, "")
	flag.Parse()

	command := os.Args[len(os.Args)-1]
	if command != "build" && command != "serve" {
		fmt.Printf(`Gozer - a fast & simple static site generator

Usage: gozer [OPTIONS] <COMMAND>

Commands:
	build	Deletes the output directory if there is one and builds the site
	serve	Builds the site and starts an HTTP server on http://localhost:8080

Options:
	-r, --root <ROOT> Directory to use as root of project (default: .)
	-c, --config <CONFIG> Path to confiruation file (default: config.xml)
`)
		return
	}

	if rootPath != "" {
		rootPath = strings.TrimSuffix(rootPath, "/") + "/"
	}

	var err error
	timeStart := time.Now()

	templates, err = template.ParseFS(os.DirFS(rootPath+"templates/"), "*.html")
	if err != nil {
		log.Fatal("Error reading templates/ directory: %s", err)
	}

	// read config.xml
	var config *Config
	config, err = parseConfig(rootPath + configFile)
	if err != nil {
		log.Fatal("Error reading configuration file at %s: %w\n", rootPath+configFile, err)
	}

	site := Site{
		SiteUrl: config.SiteUrl,
		RootDir: rootPath,
	}

	// read content
	if err := site.readContent(rootPath + "content/"); err != nil {
		log.Fatal("Error reading content/: %s", err)
	}

	// build each individual page
	for _, p := range site.pages {
		if err := site.buildPage(&p); err != nil {
			log.Warn("Error processing %s: %s\n", p.Filepath, err)
		}
	}

	// create XML sitemap
	if err := site.createSitemap(); err != nil {
		log.Warn("Error creating sitemap: %s\n", err)
	}

	// create RSS feed
	if err := site.createRSSFeed(); err != nil {
		log.Warn("Error creating RSS feed: %s\n", err)
	}

	// static files
	if err := copyDirRecursively(rootPath+"public/", "build/"); err != nil {
		log.Fatal("Error copying public/ directory: %s", err)
	}

	log.Info("Built site containing %d pages in %d ms\n", len(site.pages), time.Since(timeStart).Milliseconds())

	if command == "serve" {
		log.Info("Listening on http://localhost:8080\n")
		log.Fatal("Hello", http.ListenAndServe("localhost:8080", http.FileServer(http.Dir("build/"))))
	}
}
