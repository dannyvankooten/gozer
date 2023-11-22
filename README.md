# Gozer!

Gozer is a simple static site generator written in Golang.

- Converts Markdown to HTML.
- Allows you to use page-specific templates.
- Creates an XML sitemap for search engines.
- Creates an RSS feed for feed readers.

Here's a [sample website built using Gozer](https://github.com/dannyvankooten/www.dannyvankooten.com).

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

- `gozer` Builds the site into `build/`
- `gozer serve` Builds the site into `build/` and starts an HTTP server on `localhost:8080` serving the `build/` directory.

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
