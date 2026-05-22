# Dedsite

Dedsite is a personal resume and portfolio site built with a small Go backend, SQLite content storage, server-rendered templates, and HTMX navigation.

The current design is a dark terminal/TUI-inspired interface with violet borders, a profile panel, GitHub and LinkedIn contact links, and separate portfolio sections for Experience, Projects, Security, and AI.

## Tools Used

- Go: HTTP server, routing, templates, and application logic.
- SQLite: Local database for profile, sections, and portfolio entries.
- `modernc.org/sqlite`: Pure-Go SQLite driver.
- HTMX: Partial page updates for section navigation.
- HTML templates: Server-rendered pages and reusable partials.
- CSS: Custom dark terminal-style theme.

## Project Layout

```text
cmd/server/main.go        Server entrypoint
internal/db               Database models, migrations, seed data, and queries
internal/handlers         HTTP route handlers
templates                 Full-page HTML templates
templates/partials        Reusable and HTMX partial templates
static/css/site.css       Site styles
data/site.db              Local SQLite database, ignored by Git
```

## Running Locally

Install Go, then run:

```powershell
go mod tidy
$env:ADDR=':8081'
go run ./cmd/server
```

Open:

```text
http://localhost:8081
```

If that port is busy, choose another:

```powershell
$env:ADDR=':8082'
go run ./cmd/server
```

## Configuration

`ADDR` sets the server listen address. The default is `:8080`.

`DATABASE_PATH` sets the SQLite database path. The default is `data/site.db`.

## How Content Works

On startup, the app runs migrations and then seeds or syncs project defaults.

The database has three main tables:

- `profile`: Name, title, email, and summary.
- `sections`: Top-level navigation sections.
- `items`: Entries inside each section, including tags, optional links, and optional images.

During early development, some defaults in `internal/db/seed.go` are synchronized into the local database on startup. This keeps the site easy to reshape without manually deleting `data/site.db`.

To add an image to any section item, set `ImageURL` and optionally `ImageAlt` in the seed data or the matching `image_url` and `image_alt` columns in SQLite. Local project images can live under `static`, then be referenced with a URL like `/static/images/example.png`.

## HTMX Navigation

The section tabs request `/section/{slug}` with HTMX. Normal browser requests get the full page. HTMX requests get a smaller partial response that swaps the section content and updates the active tab state.

Template changes require restarting the Go server because templates are parsed on startup. CSS changes usually only need a browser refresh.
