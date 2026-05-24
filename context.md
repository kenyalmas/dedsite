# Current Context

## Active Branch
- `feature/awareness-terminal-tab`

## Current Goal
- Add a terminal-themed awareness page that demonstrates what websites can infer from normal browser requests.
- Ensure the page loads reliably in local development on port `8081`.

## Confirmed Working URLS (May 24, 2026)
- Home: `http://localhost:8081/`
- Awareness page: `http://localhost:8081/awareness`

## Important Routes
- `GET /` -> Home
- `GET /awareness` -> Awareness demo page
- `GET /static/*` -> Shared CSS/JS

## Key Files
- `cmd/server/main.go` (local dev router)
- `internal/handlers/handlers.go` (Awareness handler)
- `templates/awareness.html` (awareness tab UI)
- `templates/home.html` (link to awareness tab)
- `static/css/site.css` (terminal theme + awareness styling)
- `netlify/functions/server/main.go` (Netlify runtime router parity)

## Root Cause of "page doesn't load"
- Port `8081` was occupied by another process, so the intended server build could not bind to that port.
- Result: requests were hitting a different process and returned `404`.

## Fix Applied
- Stopped the process bound to `8081`.
- Restarted the app on `8081` with `go run ./cmd/server --port 8081`.
- Verified `GET /awareness` now returns HTTP `200`.

## Collaboration Preference (User Instruction)
- If the user has to ask for the same outcome more than once, record it in this file and treat it as guidance for future steps in the same project.

## Repeat-Request Notes
- 2026-05-24: User asked to remove specific awareness fields, then had to ask again because stale server output was still shown. Action taken: restarted server on `8081`, re-verified removed fields were absent, and documented this preference.

## Working Commands
```powershell
# Start local server on 8081
go run ./cmd/server --port 8081

# Optional: sync assets to Netlify function bundle
go run ./cmd/sync-netlify-assets
```
