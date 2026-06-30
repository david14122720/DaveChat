# Delta for API Gateway

## ADDED Requirements

### Requirement: New Routes

The system MUST add the following routes to the Echo router:

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/auth/refresh` | No | Refresh token rotation |
| `POST` | `/api/conversations/{id}/read` | Yes | Mark messages as read |
| `GET` | `/api/conversations/{id}/messages/search` | Yes | FULLTEXT search |
| `GET` | `/api/calls` | Yes | Call history |
| `GET` | `/assets/*` | No* | Static files (embed, production only) |
| `GET` | `/*` | No* | SPA fallback (production only) |

(* Routes registered only when `DEV_MODE=false`)

### Requirement: Echo Error Handler

The system MUST register a custom `HTTPErrorHandler` on the Echo instance that returns consistent JSON error responses.

#### Scenario: 404 returns JSON error

- GIVEN a request to `/api/nonexistent`
- WHEN Echo's router finds no matching route
- THEN response is HTTP 404 with `{"error":{"code":"NOT_FOUND","message":"Route not found"}}`

#### Scenario: Echo validation error returns 400

- GIVEN a request with invalid JSON body
- WHEN Echo's binder fails
- THEN response is HTTP 400 with `{"error":{"code":"INVALID_INPUT","message":"..."}}`

## MODIFIED Requirements

### Requirement: CORS for Dev Mode

(Previously: chi CORS middleware with `AllowedOrigins` from config)

The system MUST configure Echo's CORS middleware to allow cross-origin requests in dev mode.

#### Scenario: DEV_MODE=true enables Vite CORS

- GIVEN `DEV_MODE=true`
- WHEN the browser sends OPTIONS preflight with `Origin: http://localhost:5173`
- THEN response includes `Access-Control-Allow-Origin: http://localhost:5173`

#### Scenario: DEV_MODE=false has no CORS headers

- GIVEN `DEV_MODE` is unset
- WHEN same-origin request is made
- THEN no CORS headers are sent (static serving, same origin)

### Requirement: Unified Port

(Previously: chi router on port 8080)

The system MUST serve REST API, WebSocket, and optionally static files on a single port (8080) using Echo v4 router.

#### Scenario: All routes on same port with Echo

- GIVEN the Go server is running with Echo on port 8080
- WHEN `GET /api/profiles` is called
- THEN it is routed to the Echo handler
- WHEN `ws://host/ws` is connected
- THEN WebSocket upgrade succeeds via `echo.WrapHandler`
- WHEN `GET /` is called in production mode
- THEN embedded `index.html` is served

### Requirement: Frontend api.js Client

(Previously: client uses `VITE_API_URL` env var)

The client SHOULD omit the API base URL when served from the embedded binary (same origin). In dev mode, `VITE_API_URL` still points to `http://localhost:8080`.

#### Scenario: Embedded frontend uses relative URLs

- GIVEN the frontend is served from the embedded binary
- WHEN `api.js` sends a request
- THEN the base URL is empty string (relative `fetch("/api/...")`)
- AND `Authorization` header is still sent
