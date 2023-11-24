package main

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

type Site struct {
	pages []Page
	posts []Page

	Title   string `toml:"title"`
	SiteUrl string `toml:"url"`
	RootDir string
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
	path = strings.TrimSuffix(path, ".html")
	path = strings.TrimSuffix(path, "index")

	filename := filepath.Base(path)
	if len(filename) > 11 && filename[4] == '-' && filename[7] == '-' && filename[10] == '-' {
		date, err := time.Parse("2006-01-02", filename[0:10])
		if err == nil {
			return path[0:len(path)-len(filename)] + filename[11:] + "/", date
		}
	}

	if path != "" && path[len(path)-1] != '/' {
		path += "/"
	}

	return path, time.Time{}
}

func parseFrontMatter(p *Page) error {
	fh, err := os.Open(p.Filepath)
	if err != nil {
		return err
	}
	defer fh.Close()

	buf := make([]byte, 1024)
	n, err := fh.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:n]
	if !bytes.HasPrefix(buf, frontMatter) {
		return nil
	}

	// strip front matter prefix
	buf = buf[3:]

	// find pos of closing front matter
	pos := bytes.Index(buf, frontMatter)
	if pos == -1 {
		return fmt.Errorf("missing closing front-matter identifier in %s", p.Filepath)
	}

	return toml.Unmarshal(buf[:pos], p)
}

func (p *Page) ParseContent() (string, error) {
	fileContent, err := os.ReadFile(p.Filepath)
	if err != nil {
		return "", err
	}

	// Skip front matter
	if len(fileContent) > 6 {
		pos := bytes.Index(fileContent[3:], frontMatter)
		if pos > -1 {
			fileContent = fileContent[pos+6:]
		}
	}

	// If source file has HTML extension, return content directly
	if strings.HasSuffix(p.Filepath, ".html") {
		return string(fileContent), nil
	}

	// Otherwise, parse as Markdown
	var buf2 strings.Builder
	if err := md.Convert(fileContent, &buf2); err != nil {
		return "", err
	}
	return buf2.String(), nil
}

func (s *Site) buildPage(p *Page) error {
	content, err := p.ParseContent()
	if err != nil {
		return err
	}

	dest := "build/" + p.UrlPath + "index.html"
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	fh, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fh.Close()

	tmpl := templates.Lookup(p.Template)
	if tmpl == nil {
		return fmt.Errorf("invalid template name: %s", p.Template)
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

func (s *Site) AddPageFromFile(file string) error {
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	urlPath, datePublished := parseFilename(file, s.RootDir)

	p := Page{
		Filepath:      file,
		UrlPath:       urlPath,
		Permalink:     s.SiteUrl + urlPath,
		DatePublished: datePublished,
		DateModified:  info.ModTime(),
		Template:      "default.html",
	}

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
	defer measure("readContent")()

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
	defer measure("createSitemap")()

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
			Loc:     p.Permalink,
			LastMod: p.DateModified.Format(time.RFC3339),
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

	if _, err := wr.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><?xml-stylesheet type="text/xsl" href="/sitemap.xsl"?>`)); err != nil {
		return err
	}
	if err := xml.NewEncoder(wr).Encode(env); err != nil {
		return err
	}

	// copy xml stylesheet
	if err := os.WriteFile("build/sitemap.xsl", sitemapXSL, 0655); err != nil {
		return err
	}

	return nil
}

func (s *Site) createRSSFeed() error {
	defer measure("createRSSFeed")()

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
			log.Warn("error parsing content of %s: %s", p.Filepath, err)
			continue
		}

		items = append(items, Item{
			Title:       p.Title,
			Link:        p.Permalink,
			Description: pageContent,
			PubDate:     p.DatePublished.Format(time.RFC1123Z),
			GUID:        p.Permalink,
		})
	}

	feed := Feed{
		Version: "2.0",
		Atom:    "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:         s.Title,
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

	if _, err := wr.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`)); err != nil {
		return err
	}

	if err := xml.NewEncoder(wr).Encode(feed); err != nil {
		return err
	}

	return nil
}

// func to calculate and print execution time
func measure(name string) func() {
	start := time.Now()
	return func() {
		log.Info("%s execution time: %v\n", name, time.Since(start))
	}
}

func parseConfig(s *Site, file string) error {
	_, err := toml.DecodeFile(file, s)
	if err != nil {
		return err
	}

	// ensure site url has trailing slash
	if !strings.HasSuffix(s.SiteUrl, "/") {
		s.SiteUrl += "/"
	}

	return nil
}

func main() {
	configFile := "config.toml"
	rootPath := ""

	// parse flags
	flag.StringVar(&configFile, "config", configFile, "")
	flag.StringVar(&configFile, "c", configFile, "")
	flag.StringVar(&rootPath, "root", rootPath, "")
	flag.StringVar(&rootPath, "r", rootPath, "")
	flag.Parse()

	command := os.Args[len(os.Args)-1]
	if command != "build" && command != "serve" && command != "new" {
		fmt.Printf(`Gozer - a fast & simple static site generator

Usage: gozer [OPTIONS] <COMMAND>

Commands:
	build	Deletes the output directory if there is one and builds the site
	serve	Builds the site and starts an HTTP server on http://localhost:8080
	new     Creates a new site structure in the given directory

Options:
	-r, --root <ROOT> Directory to use as root of project (default: .)
	-c, --config <CONFIG> Path to configuration file (default: config.toml)
`)
		return
	}

	// ensure rootPath has a trailing slash
	if rootPath != "" && !strings.HasSuffix(rootPath, "/") {
		rootPath += "/"
	}

	if command == "new" {
		if err := createDirectoryStructure(rootPath); err != nil {
			log.Fatal("Error creating site structure: ", err)
		}
		return
	}

	buildSite(rootPath, configFile)

	if command == "serve" {
		log.Info("Listening on http://localhost:8080\n")
		log.Fatal("Hello", http.ListenAndServe("localhost:8080", http.FileServer(http.Dir("build/"))))
	}
}

func createDirectoryStructure(rootPath string) error {

	if err := os.Mkdir(rootPath+"content", 0755); err != nil {
		return err
	}
	if err := os.Mkdir(rootPath+"templates", 0755); err != nil {
		return err
	}
	if err := os.Mkdir(rootPath+"public", 0755); err != nil {
		return err
	}

	// create configuration file
	if err := os.WriteFile(rootPath+"config.toml", []byte("url = \"http://localhost:8080\"\ntitle = \"My website\"\n"), 0655); err != nil {
		return err
	}

	// create default template
	if err := os.WriteFile(rootPath+"templates/default.html", []byte("<!DOCTYPE html>\n<head>\n\t<title>{{ .Title }}</title>\n</head>\n<body>\n{{ .Content }}\n</body>\n</html>"), 0655); err != nil {
		return err
	}

	// create homepage
	if err := os.WriteFile(rootPath+"content/index.md", []byte("+++\ntitle = \"Gozer!\"\n+++\n\nWelcome to my website.\n"), 0655); err != nil {
		return err
	}

	return nil
}

func buildSite(rootPath string, configFile string) {
	var err error
	timeStart := time.Now()

	templates, err = template.ParseFS(os.DirFS(rootPath+"templates/"), "*.html")
	if err != nil {
		log.Fatal("Error reading templates/ directory: %s", err)
	}

	// read config.xml
	site := &Site{
		RootDir: rootPath,
	}

	if err := parseConfig(site, rootPath+configFile); err != nil {
		log.Fatal("Error reading configuration file at %s: %w\n", rootPath+configFile, err)
	}

	// read content
	if err := site.readContent(rootPath + "content/"); err != nil {
		log.Fatal("Error reading content/: %s", err)
	}

	var wg sync.WaitGroup

	// build each individual page
	wg.Add(len(site.pages))
	for _, p := range site.pages {
		go func(p Page) {
			defer wg.Done()

			if err := site.buildPage(&p); err != nil {
				log.Warn("Error processing %s: %s\n", p.Filepath, err)
			}
		}(p)
	}
	wg.Wait()

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
}
