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
	build	Deletes the output directory if there is one and builds the site
	serve	Builds the site and starts an HTTP server on http://localhost:8080

Options:
	-r, --root <ROOT> Directory to use as root of project (default: .)
	-c, --config <CONFIG> Path to confiruation file (default: config.xml)
```


## Content files

Each file in your `content/` directory should end in `.md` and have front matter specifying the page title:

```md
+++
title = "My page title"
+++

Page content here.
```

The default template for every page is `default.html`. You can override it by setting the `template` variable in your front matter.

```md
+++
title = "My page title"
template = "special-page.html"
+++

Page content here.
```


## License

Gozer is open-sourced under the MIT license.
