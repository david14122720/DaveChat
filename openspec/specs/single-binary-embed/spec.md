# Single Binary Embed Specification

## Purpose

Embed the Vite production build inside the Go binary via `//go:embed`, enabling zero-dependency deployment. The server serves its own frontend in production mode.

## Requirements

### Requirement: Embed Build Output

The system MUST embed `client/dist/*` into the Go binary using `//go:embed`.

```go
//go:embed client/dist/*
var staticFiles embed.FS
```

#### Scenario: Build output is accessible at compile time

- GIVEN `client/dist/` contains `index.html`, `assets/index-abc123.js`, `assets/index-xyz789.css`
- WHEN `go build ./...` runs
- THEN the binary compiles without errors
- AND `staticFiles` contains all files from `client/dist/`

### Requirement: DEV_MODE Switching

The system MUST check `DEV_MODE` env var to toggle between Vite dev server (CORS) and embedded static serving.

#### Scenario: DEV_MODE=true enables CORS only

- GIVEN `DEV_MODE=true`
- WHEN the server starts
- THEN Echo CORS allows `http://localhost:5173`
- AND no static file routes are registered
- AND frontend connects to Vite dev server at port 5173

#### Scenario: DEV_MODE=false serves embedded files

- GIVEN `DEV_MODE` is unset or `false`
- WHEN a browser requests `/`
- THEN the server responds with the embedded `index.html`
- AND no CORS headers are sent (same-origin)

### Requirement: Static File Serving

The system MUST serve `/assets/*` files with caching headers and provide SPA fallback for non-API routes.

#### Scenario: Hashed assets are cacheable

- GIVEN `client/dist/assets/main-abc123.js` exists
- WHEN `GET /assets/main-abc123.js` is requested
- THEN response is HTTP 200 with `Cache-Control: public, max-age=31536000, immutable`
- AND content matches the embedded file

#### Scenario: Non-file routes serve index.html (SPA fallback)

- GIVEN user navigates to `/chat` (not an API route and no file at that path)
- WHEN `GET /chat` is requested
- THEN response is HTTP 200 with the embedded `index.html` content
- AND API routes (`/api/*`, `/ws`, `/health`) are NOT intercepted

### Requirement: Vite Build Config

The system MUST configure Vite with `base: '/'` and build output to `client/dist/`.

#### Scenario: Build produces expected output

- GIVEN `client/vite.config.js` with `base: '/'`
- WHEN `npm run build` runs in `client/`
- THEN output is in `client/dist/`
- AND `client/dist/index.html` uses relative asset paths (`/assets/main-abc123.js`)
