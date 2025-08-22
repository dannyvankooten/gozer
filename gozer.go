package main

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

var templates *template.Template

//go:embed sitemap.xsl
var sitemapXSL []byte

var now = time.Now()

type Site struct {
	Pages []Page
	Posts []Page

	Title   string `toml:"title"`
	SiteUrl string `toml:"url"`
	RootDir string
}

type Page struct {
	// Title of this page
	Title string

	// Template this page uses for rendering. Defaults to "default.html".
	Template string

	// Time this page was published (parsed from file name).
	DatePublished time.Time

	// Time this page was last modified (from filesystem).
	DateModified time.Time

	// The full URL to this page (incl. site URL)
	Permalink string

	// URL path for this page, relative to site URL
	UrlPath string

	// Path to source file for this page, relative to content root
	Filepath string

	// Whether the page should be published or not
	Draft bool `toml:"draft"`

	// A list of tags associated with the page
	Tags []string `toml:"tags"`
}

// parseFilename parses the URL path and optional date component from the given file path
func parseFilename(path string, rootDir string) (string, time.Time) {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, rootDir+"content/")
	path = strings.TrimSuffix(path, ".md")
	path = strings.TrimSuffix(path, ".dj")
	path = strings.TrimSuffix(path, ".html")
	path = strings.TrimSuffix(path, "index")

	filename := filepath.Base(path)
	if len(filename) > 11 && filename[4] == '-' && filename[7] == '-' && filename[10] == '-' {
		date, err := time.Parse("2006-01-02", filename[0:10])
		if err == nil {
			return path[0:len(path)-len(filename)] + filename[11:] + "/", date
		}
	}

	if path != "" && !strings.HasSuffix(path, "/") {
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

	switch filepath.Ext(p.Filepath) {
	default:
		fmt.Printf("Unknown file type %q (%q)\n", p.Filepath, filepath.Ext(p.Filepath))
	case ".md":
		var buf2 strings.Builder
		fmt.Printf("processing %q\n", p.Filepath)
		if err := md.Convert(fileContent, &buf2); err != nil {
			return "", err
		}
		return buf2.String(), nil
	case ".dj":
		return ConvertDjot(fileContent), nil
	case ".html":
		return string(fileContent), nil
	}
	return "", fmt.Errorf("unexpected error")

}

func (s *Site) buildPage(p *Page) error {
	content, err := p.ParseContent()
	if err != nil {
		return err
	}

	dest := filepath.Join("build", p.UrlPath, "index.html")
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

	var prev, next *Page
	for i, post := range s.Posts {
		if &post == p {
			if i > 0 {
				prev = &s.Posts[i-1]
			}
			if i < len(s.Posts)-1 {
				next = &s.Posts[i+1]
			}
		}
	}

	return tmpl.Execute(fh, map[string]any{
		"Page":  p,
		"Posts": s.Posts,
		"Pages": s.Pages,
		"Site": map[string]string{
			"Url":   s.SiteUrl,
			"Title": s.Title,
		},

		// If the page is a post, it may have a next and previous post
		// These may also be nil
		"Prev": prev,
		"Next": next,

		// Shorthand for accessing through .Page.Title / .Page.Content
		"Title":   p.Title,
		"Content": template.HTML(content),

		// Timestamp of build
		"Now": now,

		// Deprecated template variables, use .Site.Url instead
		"SiteUrl": s.SiteUrl,
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
		return fmt.Errorf("%s: %s", file, err)
	}

	s.Pages = append(s.Pages, p)

	// every page with a date is assumed to be a blog post
	if !p.DatePublished.IsZero() {
		s.Posts = append(s.Posts, p)
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
	sort.Slice(s.Posts, func(i int, j int) bool {
		return s.Posts[i].DatePublished.After(s.Posts[j].DatePublished)
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

	urls := make([]Url, 0, len(s.Pages))
	for _, p := range s.Pages {
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

	sitemapFilename := filepath.Join("build", "sitemap.xml")
	wr, err := os.Create(sitemapFilename)
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
	sitemapStylesheetFilename := filepath.Join("build", "sitemap.xsl")
	if err := os.WriteFile(sitemapStylesheetFilename, sitemapXSL, 0655); err != nil {
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
	n := len(s.Posts)
	if n > 10 {
		n = 10
	}

	items := make([]Item, 0, n)
	for _, p := range s.Posts[0:n] {
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
			Generator:     "Gozer",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Items:         items,
		},
	}

	rssFeedFilename := filepath.Join("build", "feed.xml")
	wr, err := os.Create(rssFeedFilename)
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
	showHelp := false
	listen := "localhost:8080"

	// parse flags
	flag.StringVar(&configFile, "config", configFile, "")
	flag.StringVar(&configFile, "c", configFile, "")
	flag.StringVar(&rootPath, "root", rootPath, "")
	flag.StringVar(&rootPath, "r", rootPath, "")
	flag.BoolVar(&showHelp, "help", showHelp, "")
	flag.BoolVar(&showHelp, "h", showHelp, "")
	flag.StringVar(&listen, "listen", "localhost:8080", "")
	flag.Parse()

	command := os.Args[len(os.Args)-1]
	if showHelp || (command != "build" && command != "serve" && command != "new" && command != "watch") {
		fmt.Printf(`Gozer - a fast & simple static site generator

Usage: gozer [OPTIONS] <COMMAND>

Commands:
	build	Deletes the output directory if there is one and builds the site
	serve	Builds the site and starts an HTTP server on http://localhost:8080
	watch   Builds the site and watches for file changes
	new     Creates a new site structure in the given directory

Options:
	-r, --root <ROOT> Directory to use as root of project (default: .)
	-c, --config <CONFIG> Path to configuration file (default: config.toml)
	    --listen <INTERFACE:PORT> Interface to liston on; only used with 'serve',
	             'INTERFACE' is optional. e.g. '--listen :9000'
`)
		return
	}

	if command == "new" {
		if err := createDirectoryStructure(rootPath); err != nil {
			log.Fatal("Error creating site structure: ", err)
		}
		return
	}

	buildSite(rootPath, configFile)

	if command == "serve" || command == "watch" {
		// safety is to make sure we don't let the user ^C exit while we're in the middle of rebuilding
		// a site. This only affects "watch", as ListenAndServe does its own ^C handling.
		safety := sync.Mutex{}
		// setup fsnotify watcher
		go watchDirs([]string{
			filepath.Join(rootPath, "content"),
			filepath.Join(rootPath, "public"),
			filepath.Join(rootPath, "templates"),
		}, func() {
			// prevent ^C during a build
			safety.Lock()
			buildSite(rootPath, configFile)
			safety.Unlock()
		})

		if command == "serve" {
			log.Info("Listening on http://localhost:8080\n")
			_ = http.ListenAndServe("localhost:8080", http.FileServer(http.Dir("build")))
		} else {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, os.Interrupt)
			log.Info("Watching for changes (Ctrl-C to exit)\n")
			for range ch {
				log.Info("User inturrupt... ")
				// wait for any current site builds to finish...
				safety.Lock()
				log.Info("Exiting\n")
				os.Exit(0)
			}
		}
	}
}

func createDirectoryStructure(rootPath string) error {
	for _, dir := range []string{"content", "templates", "public"} {
		if err := os.Mkdir(filepath.Join(rootPath, dir), 0755); err != nil {
			return err
		}
	}

	files := []struct {
		Name    string
		Content []byte
	}{
		{"config.toml", []byte("url = \"http://localhost:8080\"\ntitle = \"My website\"\n")},
		{"templates/default.html", []byte("<!DOCTYPE html>\n<head>\n\t<title>{{ .Title }}</title>\n</head>\n<body>\n{{ .Content }}\n</body>\n</html>")},
		{"content/index.md", []byte("+++\ntitle = \"Gozer!\"\n+++\n\nWelcome to my website.\n")},
		// TODO djot does not (yet) support front matter, and godjot does not parse it. Once the front-matter syntax is settled, this should change. +djot +frontmatter
		{"content/index.dj", []byte("+++\ntitle = \"Gozer!\"\n+++\n\nWelcome to my website.\n")},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(rootPath, f.Name), f.Content, 0655); err != nil {
			return err
		}
	}

	return nil
}

type PageGroup struct {
	Key   string
	Pages []Page
}

func buildSite(rootPath string, configFile string) {
	var err error
	timeStart := time.Now()

	temp := template.New("gozer").Funcs(template.FuncMap{
		"HasPrefix": strings.HasPrefix,
		"HasSuffix": strings.HasSuffix,
		"Contains":  strings.Contains,
		"Replace":  strings.Replace,
		// GroupByDate groups pages in the list by the Time spec, e.g. "2006",
		// "January", in reverse order
		"GroupByDate": func(pages []Page, date string) []PageGroup {
			groups := make(map[string][]Page)
			keys := make([]string, 0)
			for _, page := range pages {
				key := page.DateModified.Format(date)
				if groups[key] == nil {
					keys = append(keys, key)
					groups[key] = []Page{page}
				} else {
					groups[key] = append(groups[key], page)
				}
			}
			sort.Strings(keys)
			slices.Reverse(keys)
			rv := make([]PageGroup, len(keys))
			for i, key := range keys {
				pgs := groups[key]
				slices.Reverse(pgs)
				rv[i] = PageGroup{
					Key:   key,
					Pages: groups[key],
				}
			}
			return rv
		},
	})
	templates, err = temp.ParseGlob(filepath.Join(rootPath, "templates/*.html"))
	if err != nil {
		log.Fatal("Error reading templates/ directory: %s", err)
	}

	// read config.xml
	site := &Site{
		RootDir: rootPath,
	}

	if err := parseConfig(site, filepath.Join(rootPath, configFile)); err != nil {
		log.Fatal("Error reading configuration file at %s: %w\n", rootPath+configFile, err)
	}

	// read content
	if err := site.readContent(filepath.Join(rootPath, "content")); err != nil {
		log.Fatal("Error reading content/: %s", err)
	}

	var wg sync.WaitGroup

	// build each individual page
	for _, p := range site.Pages {
		wg.Add(1)

		go func(p Page) {
			if err := site.buildPage(&p); err != nil {
				log.Warn("Error processing %s: %s\n", p.Filepath, err)
			}

			wg.Done()
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
	if err := copyDirRecursively(filepath.Join(rootPath, "public"), "build"); err != nil {
		log.Fatal("Error copying public/ directory: %s", err)
	}

	log.Info("Built %d pages in %d ms\n", len(site.Pages), time.Since(timeStart).Milliseconds())
}
