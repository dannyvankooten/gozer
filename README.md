# Gozer! [![builds.sr.ht status](https://builds.sr.ht/~dvko/gozer.svg)](https://builds.sr.ht/~dvko/gozer?)

Gozer is a fast & simple static site generator written in Golang.

- Converts Markdown to HTML.
- Allows you to use page-specific templates.
- Creates an XML sitemap for search engines.
- Creates an RSS feed for feed readers.

Sample websites using Gozer:

- [Simplest possible example](example/)
- My personal website: [site](https://www.dannyvankooten.com/) - [source](https://git.sr.ht/~dvko/dannyvankooten.com)


## Installation

You can install Gozer by first installing a Go compiler and then running:

```sh
go install git.sr.ht/~dvko/gozer@latest
```

## Usage

Run `gozer new` to quickly generate an empty directory structure.

```txt
├── config.toml                # Configuration file
├── content                    # Posts and pages
│   └── index.md
├── public                     # Static files
└── templates                  # Template files
    └── default.html
```

Then, run `gozer build` to generate your site.

Any Markdown files placed in your `content/` directory will result in an HTML page in your build directory after running `gozer build`.

For example:

- `content/index.md` creates a file `build/index.html` so it is accessible over HTTP at `/`
- `content/about.md` creates a file `build/about/index.html` so it is accessible over HTTP at `/about/`.


## Commands

Run `gozer` without any arguments to view the help text.

```
Gozer - a fast & simple static site generator

Usage: gozer [OPTIONS] <COMMAND>

Commands:
    build   Deletes the output directory if there is one and builds the site
    serve   Builds the site and starts an HTTP server on http://localhost:8080
    new     Creates a new site structure in the given directory

Options:
    -r, --root <ROOT> Directory to use as root of project (default: .)
    -c, --config <CONFIG> Path to configuration file (default: config.toml)
```

## Content files

Each file in your `content/` directory should end in `.md` and have TOML front matter specifying the page title:

```md
+++
title = "My page title"
+++

Page content here.
```

### Templates
The default template for every page is `default.html`. You can override it by setting the `template` variable in your front matter.

```md
+++
title = "My page title"
template = "special-page.html"
+++

Page content here.
```

Templates are powered by Go's standard `html/template` package, so you can use all the [actions described here](https://pkg.go.dev/text/template#hdr-Actions).

Every template receives the following set of variables:

```
Pages       # Slice of all pages in the site
Posts       # Slice of all posts in the site (any page with a date in the filename)
Site        # Global site properties: Url, Title
Page        # The current page: Title, Permalink, UrlPath, DatePublished, DateModified
Title       # The current page title, shorthand for Page.Title
Content     # The current page's HTML content.
```

The `Page` variable is an instance of the object below:

```
type Page struct {
    // Title of this page
    Title         string

    // Template this page uses for rendering. Defaults to "default.html".
    Template      string

    // Time this page was published (parsed from file name).
    DatePublished time.Time

    // Time this page was last modified on the filesystem.
    DateModified  time.Time

    // The full URL to this page, including the site URL.
    Permalink     string

    // URL path for this page, relative to site URL
    UrlPath       string

    // Path to source file for this page, relative to content root
    Filepath      string
}
```

To show a list of the 5 most recent posts:

```gotemplate
{{ range (slice .Posts 0 5) }}
    <a href="{{ .Permalink }}">{{ .Title }}</a> <small>{{ .DatePublished.Format "Jan 02, 2006" }}</small><br />
{{ end }}
```

## Contributing

Gozer development happens on [Sourcehut](https://sr.ht/). Patches are accepted over email.

- [Code repository](https://git.sr.ht/~dvko/gozer)
- [Mailing list](https://lists.sr.ht/~dvko/gozer-devel)
- [Issue tracker](https://todo.sr.ht/~dvko/gozer-issues)

## License

Gozer is open-sourced under the MIT license.
