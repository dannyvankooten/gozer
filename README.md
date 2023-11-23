# Gozer!

Gozer is a fast & simple static site generator written in Golang.

- Converts Markdown to HTML.
- Allows you to use page-specific templates.
- Creates an XML sitemap for search engines.
- Creates an RSS feed for feed readers.

Sample websites using Gozer:

- [Simplest possible example](example/)
- [My personal website](https://github.com/dannyvankooten/www.dannyvankooten.com)

## Directory structure

Gozer expects a certain directory structure in order to build your site correctly.

```txt
content/        # Your posts and pages
--- index.md    # Will be generated at build/index.html
--- about.md    # Will be generated at build/about/index.html
templates/      # Your Go templates
public/         # Any static files
config.xml      # Configuration file
```

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
    -c, --config <CONFIG> Path to confiruation file (default: config.xml)
```

Run `gozer new` in an empty directory to quickly create the directory structure for a basic site.

## Content files

Each file in your `content/` directory should end in `.md` and have front matter specifying the page title:

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
SiteUrl     # The root URL of the site
Page        # The current page
Title       # The current page title. Can also be access via Page.Title
Content     # The current page's HTML content.
```

For example, to show a list of the 5 most recent posts:

```gotemplate
{{ range (slice .Posts 0 5) }}
    <a href="{{ .Permalink }}">{{ .Title }}</a> <small>{{ .DatePublished.Format "Jan 02, 2006" }}</small><br />
{{ end }}
```

## License

Gozer is open-sourced under the MIT license.
