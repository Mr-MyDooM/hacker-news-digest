# Hacker News Digest

AI-powered summaries of top Hacker News stories. Stay informed with concise digests of the best technical discussions, curated daily.

Uses Gemini and OpenRouter APIs to generate article summaries.

## Features

* AI-generated summaries of top HN stories
* Static site hosted at [HackerNews.mrityunjay.dev](https://HackerNews.mrityunjay.dev/)
* Automatic daily updates
* RSS feed support

## Quick Start

```bash
# Build binary
make build

# Initialize database
make initdb

# Generate home page
./hndigest home

# Generate daily pages
./hndigest daily
```

## Project Structure

- `main.go` - Entry point for static site generation
- `hn/` - Article processing, summarization, HN scraper, Algolia API
- `config/config.go` - Configuration and thresholds
- `db/` - SQLite caching (no CGO, via modernc.org/sqlite)
- `llm/` - AI clients (Gemini, OpenAI, OpenRouter)
- `extractor/` - HTML content extraction and image handling
- `template/` - Go html/template rendering + Atom feed
- `static/` - CSS/JS assets
- `output/` - Generated static site

## License

GPLv3
