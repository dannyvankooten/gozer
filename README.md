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
templates/      # Your Go templates
public/         # Any static files
config.xml      # Configuration file
```

When running `gozer` in the root directory, your HTML site is built in the `build/` directory.

You can run `gozer serve` to build your site and start an HTTP server serving the `build/` directory locally.

## License

Gozer is open-sourced under the MIT license.
