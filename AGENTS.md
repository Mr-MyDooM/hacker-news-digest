# Hacker News Digest Agent Guide

This project was ported from Python to Go (May 2026).

## Essential Commands

**Generate site**: `make gh_home_page` (builds binary + creates static HTML in output/)
**Generate daily**: `make gh_daily_page` (creates dated daily pages using Algolia API)
**Build binary**: `go build -o hndigest .`
**Run tests**: `go test ./...`
**Reset DB**: `rm -f hackernews.db`
**Format**: `make format`
**Vet**: `make vet`

## Project Structure (Go)

- **Entry point**: `main.go` (CLI for static generation)
- **Config**: `config/config.go` (env vars, API keys, thresholds)
- **Database**: `db/` (no-CGO SQLite via modernc.org/sqlite)
  - `db.go` - setup, Model enum, ExpireSummaries
  - `summary.go` - summary cache CRUD
  - `image.go` - image GC
  - `translation.go` - translation cache
- **HN**: `hn/`
  - `parser.go` - HackerNewsParser (front page scraper via goquery)
  - `algolia.go` - Algolia API client for historical stories
  - `news.go` - News model, summarization orchestration
  - `types.go` - shared types
- **LLM**: `llm/`
  - `openai.go` - OpenAI + OpenRouter client (OpenAI-compatible API)
  - `gemini.go` - Gemini client
- **Extractor**: `extractor/`
  - `extractor.go` - HTML content extraction (readability via goquery)
  - `image.go` - WebImage fetch, validation, caching
- **Templates**: `template/`
  - `render.go` - Go html/template rendering + Atom feed
  - `base.gohtml` - HTML template (embedded via embed.FS)
- **Static**: `static/` (CSS/JS/fonts - same as before)
- **Output**: `output/` (generated static site - DO NOT EDIT DIRECTLY)

## Key Workflows

1. **Development**:
   - Edit code -> `make build` -> `./hndigest home` -> verify in `output/`

2. **Deployment**:
   - `make gh_home_page` -> binary + static files in output/

3. **Testing**:
   - `go test ./...`
   - No integration tests; site generation serves as integration test

## Architecture

- Single Go binary (~18MB vs 1.2GB Python Docker image)
- No CGO (pure Go SQLite via modernc.org/sqlite)
- goroutine-native concurrency for parallel article processing
- Embed.FS for templates compiled into binary
- Static files copied at generation time

## Production

- **Host**: (your server)
- **Path**: `/srv/mj/docker/hacker-news/`
- **Container**: `hn-digest` (image: `hacker-news-hn-digest`)
- **Nginx**: on host (not Docker), serves from `output/` on port 443
- **Nginx config**: `nginx-hn.conf` in repo root, deployed to `/etc/nginx/sites-available/hn.conf` (symlinked in sites-enabled)
- **Site**: https://HackerNews.mrityunjay.dev
- **Deploy**: `ssh <host>` -> `cd /srv/mj/docker/hacker-news && docker compose -f docker-compose.hn.yml up -d --build`
- **Static assets**: `main.go` copies `static/` -> `output/static/` during generation

## Anti-Scraping

- **Rate limiting**: 1 req/min per real IP (uses `CF-Connecting-IP` header, falls back to `$remote_addr`)
- **Nginx**: `limit_req zone=scrape burst=3 nodelay` on `/`; static/image paths exempted
- **User-agent blocking**: known scrapers get 403 (defense in depth, not primary)
- **Security headers**: CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy
- **robots.txt**: blocks GPTBot, Claude-Web, CCBot, ChatGPT-User, Google-Extended, anthropic-ai, Bytespider
- **Hidden files**: `.db`, `.sqlite`, `.bak`, `.log`, `.env`, dotfiles denied at nginx level
- **Cloudflare**: orange cloud already proxying; add Bot Fight Mode + Browser Integrity Check for additional protection

## Jules (AI Agent) Workflow

Jules is an AI coding agent that generates PRs. This section is the single source of
truth for Jules. Do NOT rely on `.jules/` — that directory is gitignored scratchpad and
is not present in the repo.

### Mandatory Pre-Flight Checklist (run BEFORE any work)

Jules MUST complete every step before writing code or opening a PR. If any step finds
the work already exists, STOP and report the existing work instead of opening a PR.

1. `gh pr list --state open` — is there already an OPEN PR for this feature/category?
   If yes, do NOT open another. Comment on or update the existing one.
2. `gh pr list --state merged --limit 30` — was an equivalent change already MERGED?
   If yes, the work is done; report it and stop.
3. `git log --oneline -20` and diff the target files — is the improvement already in
   `main`? If the optimization/fix/UX change is present, stop.
4. Read the target source files — confirm the problem still exists before "fixing" it.

### Hard Rules

- **One open PR per category at a time.** Never open a second PR for the same category
  (`palette`/`bolt`/`sentinel`) while one is open or merged. Duplicates are closed
  without review.
- If a solution exists in ANY form (merged, open PR, already in `main`, existing code),
  Jules must NOT create a duplicate PR — report the existing work instead.
- Verify the branch compiles before pushing: `go build ./...` and `go test ./...`.
- Do not commit `.jules/` scratchpad files (gitignored — keep it that way).

> NOTE: Recurring/scheduled Jules tasks are the common cause of duplicates — each run
> starts blind from `main`. The checklist above is what makes a run aware of prior work
> and is mandatory on every run, scheduled or not.

**Categories**:
- `palette` = UI/UX changes
- `bolt` = performance optimizations
- `sentinel` = security fixes

## Important Notes

- Requires OPENAI_API_KEY in env for AI summarization
- Requires GEMINI_API_KEY for Gemini fallback
- DB caches articles to avoid re-processing
- `output/` must NOT be edited directly (overwritten on rebuild)
- Score thresholds control summarization method:
  - OpenAI: score >= threshold (via OPENAI_SCORE_THRESHOLD, default 20)
  - Gemini: score >= threshold
  - OpenRouter: score >= 10
  - Otherwise: prefix-only summary
- LLM clients are initialised in hn.Init() - must be called before PullContent()
- PDF extraction via ledongthuc/pdf; content-type + magic byte detection
- Security: CSP/referrer meta tags, restrictive robots.txt (blocks AI scrapers)
