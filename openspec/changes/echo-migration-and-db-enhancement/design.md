# Design: Echo Migration + DB Enhancements

## Technical Approach

Three-phase migration from Chi v5 → Echo v4, additive DB schema, and single-binary embed. Each phase is independently revertible. Phase 1 is purely additive SQL. Phase 2 replaces the HTTP router layer keeping handler logic intact. Phase 3 adds embed and static serving. WebSocket remains on gorilla/websocket via `echo.WrapHandler`.

## 1. Architecture: Echo Request Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│  echo.New()                                                │
│  ├── Global middleware stack (Logger, Recover, CORS)        │
│  ├── HTTPErrorHandler (custom JSON format)                 │
│  └── Route groups: /api public/protected                   │
└─────────────────────────────────────────────────────────────┘
                           │
                    HTTP Request
                           │
                           ▼
              ┌──────────────────────┐
              │  echo.Context         │  ← c.Request(), c.Response(), c.Param(), c.QueryParam()
              │  c.Bind(&req)         │  ← JSON body binding
              │  c.Get("user_id")     │  ← values from middleware
              │  c.JSON(code, data)   │  ← response (always JSON)
              └──────────────────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
     Public routes               Protected routes
     POST /auth/register        ┌───────────────────┐
     POST /auth/login           │ AuthMiddleware     │
     POST /auth/refresh         │ c.Set("user_id")   │
                                └───────────────────┘
                                          │
                              ┌───────────┴───────────┐
                              ▼                       ▼
                        REST handlers           WebSocket
                        (Echo Context)          /api/ws
                                                echo.WrapHandler(
                         New handlers:          http.Handler)
                         /search                │
                         /read                  │
                         /call-logs        gorilla/websocket
                                              upgrader + Hub
```

**Dev mode**: frontend on Vite `:5173`, CORS allows `localhost:5173`, no static serving.

**Prod mode**: Echo serves `/assets/*` from embedded `client/dist` with `Cache-Control: public, max-age=31536000` (immutable hashed files). SPA fallback serves `index.html` for any non-asset route matching neither API nor health.

## 2. Architecture Decisions

| Decision | Options | Choice | Rationale |
|----------|---------|--------|-----------|
| Router | Keep Chi vs Echo v4 vs Gin | **Echo v4** | Proposal aligns with Echo; Echo's `c.Bind()` + `c.JSON()` reduce boilerplate vs Chi; built-in `HTTPErrorHandler` replaces ad-hoc error responses; `WrapHandler` bridges gorilla/websocket without rewrite |
| Refresh token storage | JWT blacklist vs DB table | **DB table `refresh_tokens`** | Rotation requires token hashing + expiry per backend — cannot do with JWT alone. DB row doubles as revocation list |
| Read receipts storage | Unread count vs per-message | **Per-message `read_receipts`** | Proposal requires per-message granularity for future UI. Unread count is derived: `COUNT(*) FROM messages m LEFT JOIN read_receipts rr ... WHERE rr IS NULL` |
| FULLTEXT search | LIKE + INDEX vs FULLTEXT | **FULLTEXT + BOOLEAN MODE** | `LIKE '%term%'` does not scale beyond a few hundred messages. FULLTEXT with BOOLEAN MODE supports relevance ranking and partial word matches |
| Static file serving | nginx sidecar vs Go embed | **Go embed** | Eliminates nginx dependency. Binary is self-contained. ~2MB added. Vite's hashed filenames make cache headers safe |
| WebSocket with Echo | gorilla/websocket via WrapHandler vs nhooyr.io/websocket | **gorilla via WrapHandler** | Zero changes to existing WG handler. Echo provides `echo.WrapHandler()` for exactly this case |
| Dev/Prod switching | Build tags vs env var | **`DEV_MODE` env var** | Simpler than build tags. Same binary can run both modes. Defaults to prod |

## 3. Data Flow

```
                              ┌──────────────┐
                              │   MariaDB     │
                              │  (via DSN)    │
                              └──────┬───────┘
                                     │ *sql.DB
                           ┌─────────┴─────────┐
                           │   Handler struct   │
                           │   { DB *sql.DB }   │
                           └─────────┬─────────┘
                                     │
           ┌─────────────────────────┼─────────────────────┐
           │                         │                     │
           ▼                         ▼                     ▼
   POST /auth/refresh       POST convs/:id/read     GET convs/:id/messages/search
           │                         │                     │
   ┌───────┴───────┐         ┌───────┴───────┐     ┌───────┴───────┐
   │ refresh_tokens│         │ read_receipts │     │ messages      │
   │ 1. hash token │         │ (message_id,  │     │ MATCH AGAINST │
   │ 2. revoke old │         │  user_id) ON  │     │ IN BOOLEAN    │
   │ 3. issue new  │         │  DUPLICATE KEY│     │ MODE + conv   │
   └───────────────┘         └───────────────┘     │ filter        │
                                                   └───────────────┘
```

**Refresh token rotation flow**:
```
POST /auth/refresh { refresh_token }
  → Hash incoming token
  → SELECT FROM refresh_tokens WHERE token_hash = ? AND expires_at > NOW()
  → DELETE from refresh_tokens WHERE id = ?  (rotation — consume old)
  → Generate new JWT + new refresh token
  → INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
  → Return { access_token, refresh_token }
```

## 4. File Changes

| File | Action | Description |
|------|--------|-------------|
| `infra/mariadb/init.sql` | Modify | Add `read_receipts`, `refresh_tokens`, `call_logs` tables + FULLTEXT index on `messages.content` |
| `server/go.mod` | Modify | Remove chi dependencies; add `github.com/labstack/echo/v4 v4.13` |
| `server/cmd/main.go` | Rewrite | Replace `chi.NewRouter()` → `echo.New()`. Add groups, middleware, embed init, static routes. New handler registrations |
| `server/internal/config/config.go` | Modify | Remove `AllowedOrigins`; replace with per-env CORS. Add `DevMode` bool |
| `server/internal/handlers/auth.go` | Modify | Convert `Register`, `Login`, `Me` to Echo pattern. Add `Refresh` handler. `generateJWT` returns also `refreshToken` |
| `server/internal/handlers/profile.go` | Modify | Convert `List`, `Get` to Echo pattern. Replace `chi.URLParam` → `c.Param` |
| `server/internal/handlers/conversation.go` | Modify | Convert `List`, `Create`, `Get` to Echo pattern |
| `server/internal/handlers/message.go` | Modify | Convert `List`, `Create` to Echo pattern. Add `Search`, `MarkRead` handlers |
| `server/internal/handlers/helpers.go` | Modify | Keep `errorResponse` type. Rename `writeJSON` internally; Echo handlers use `c.JSON` directly. `writeError` becomes Echo-style |
| `server/internal/handlers/call_log.go` | **Create** | New handler: `CallLogHandler` with `List` and `Create` for call history |
| `server/internal/middleware/auth.go` | Modify | Convert from `func(http.Handler) http.Handler` → `echo.MiddlewareFunc`. Use `c.Set("user_id", ...)` instead of context.WithValue |
| `server/internal/websocket/handler.go` | Modify | Add `HandleEchoWebSocket` wrapper that extracts `userID` from Echo context, then calls existing `HandleWebSocket` flow |
| `client/vite.config.js` | Modify | Add `base: '/'` and `build.outDir: 'dist'`. Remove `server.strictPort` (optional) |
| `client/Dockerfile` | **Create** | Multi-stage: `node:20` build → `/app/client/dist`, then `golang:1.25` build → copy dist into server binary |

## 5. API Endpoints — Request/Response Schemas

### POST /api/auth/refresh

**Request:**
```json
{ "refresh_token": "uuid-string" }
```

**Response 200:**
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "new-uuid",
  "expires_in": 604800
}
```

**Response 401:**
```json
{ "error": { "code": "INVALID_REFRESH", "message": "Refresh token expired or already used" } }
```

### POST /api/conversations/:id/read

**Request:** (no body — reads latest messages up to receipt)
```json
{ "last_message_id": "msg-uuid" }
```

**Response 200:**
```json
{ "status": "ok", "read_count": 3 }
```

Logic: `INSERT IGNORE INTO read_receipts (message_id, user_id) SELECT id, ? FROM messages WHERE conversation_id = ? AND id <= ? AND sender_id != ?`

### GET /api/conversations/:id/messages/search?q=termino

**Response 200:**
```json
{
  "results": [
    {
      "id": "msg-uuid",
      "conversation_id": "conv-uuid",
      "sender_id": "user-uuid",
      "content": "...message with termino...",
      "created_at": "2026-06-29T12:00:00Z"
    }
  ],
  "total": 5
}
```

Query: `SELECT ... FROM messages WHERE conversation_id = ? AND MATCH(content) AGAINST(? IN BOOLEAN MODE) ORDER BY created_at DESC LIMIT 50`

### GET /api/call-logs

**Response 200:**
```json
[
  {
    "id": "call-uuid",
    "caller_id": "user-uuid",
    "callee_id": "user-uuid",
    "type": "audio",
    "status": "completed",
    "started_at": "2026-06-29T12:00:00Z",
    "ended_at": "2026-06-29T12:05:00Z",
    "duration_secs": 300
  }
]
```

### POST /api/call-logs

**Request:**
```json
{
  "callee_id": "user-uuid",
  "type": "audio",
  "status": "completed",
  "started_at": "2026-06-29T12:00:00Z",
  "ended_at": "2026-06-29T12:05:00Z",
  "duration_secs": 300
}
```

**Response 201:** (full record with generated id)

## 6. DB Migration Strategy

Migration script `infra/mariadb/init.sql` is additive — appending new tables and index:

```sql
-- Phase 1 additions — append to existing init.sql

CREATE TABLE IF NOT EXISTS read_receipts (
    message_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    read_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    INDEX idx_user (user_id),
    INDEX idx_hash (token_hash),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS call_logs (
    id CHAR(36) PRIMARY KEY,
    caller_id CHAR(36) NOT NULL,
    callee_id CHAR(36) NOT NULL,
    type ENUM('audio','video') NOT NULL,
    status ENUM('missed','completed','rejected','cancelled','busy') NOT NULL,
    started_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    ended_at DATETIME(3) NULL,
    duration_secs INT UNSIGNED DEFAULT 0,
    INDEX idx_caller (caller_id),
    INDEX idx_callee (callee_id),
    FOREIGN KEY (caller_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE messages ADD FULLTEXT INDEX ft_messages_content (content);
```

**Migration for existing DB**: Docker recreates the container from init.sql. For prod DBs running already, run the `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE` statements manually. Each statement is idempotent.

## 7. Error Handling Design

```go
func customHTTPErrorHandler(err error, c echo.Context) {
    // Default: 500
    code := http.StatusInternalServerError
    msg := "Internal Server Error"

    switch e := err.(type) {
    case *echo.HTTPError:
        code = e.Code
        msg = fmt.Sprintf("%v", e.Message)
    case *json.SyntaxError:
        code = http.StatusUnprocessableEntity
        msg = "Invalid JSON body"
    }

    // Don't log 4xx client errors, do log 5xx
    if code >= 500 {
        c.Logger().Errorf("HTTP %d: %v", code, err)
    }

    c.JSON(code, errorResponse{
        Error: struct {
            Code    string `json:"code"`
            Message string `json:"message"`
        }{
            Code:    http.StatusText(code),
            Message: msg,
        },
    })
}
```

**Handler errors** use explicit `c.JSON(code, errorResponse{...})` — not `echo.NewHTTPError` — to keep the `code` + `message` domain field naming consistent with the existing client expectations.

**Panic handling**: Echo's `middleware.Recover()` catches panics and returns a `500` JSON response via the custom error handler.

## 8. Security Design — JWT + Refresh Token Rotation

| Aspect | Mechanism |
|--------|-----------|
| Access token | JWT (HS256), 7-day expiry. Claims: `sub` (userID), `iat`, `exp` |
| Refresh token | Random UUID (v4). Stored as SHA-256 hash in DB. 30-day expiry |
| Rotation | Each refresh: DELETE old token, INSERT new. Old token is single-use only |
| Token storage (client) | `localStorage` for access token (existing). Refresh token stored in same localStorage (simplicity for MVP; HttpOnly cookie preferred for higher security) |
| Replay protection | Hash comparison on read + row-level DELETE. If a compromised refresh token is reused, it won't match the (already-deleted) hash |
| JWT middleware | Extracts `Bearer` from `Authorization` header, parses JWT, sets `c.Set("user_id", userID)` for downstream handlers |

```
Client                          Server
  │                                │
  │  POST /auth/login              │
  │───────────────────────────────>│
  │  { access_token, refresh }     │
  │<───────────────────────────────│
  │                                │
  │  (access token expires)        │
  │                                │
  │  POST /auth/refresh            │
  │  { refresh_token }             │
  │───────────────────────────────>│
  │  ✓ Hash matches DB row?        │
  │  ✓ DELETE old row (rotate)     │
  │  ✓ INSERT new refresh token    │
  │  { access_token, new_refresh } │
  │<───────────────────────────────│
```

## 9. Static File Serving Strategy

| File type | Cache header | Route | Serving mode |
|-----------|-------------|-------|--------------|
| `/assets/*.js` (Vite hashed) | `public, max-age=31536000, immutable` | `e.GET("/assets/*", ...)` | From embed FS, stripped prefix |
| `/assets/*.css` (Vite hashed) | Same | Same | Same |
| `/assets/*` (images/fonts) | Same | Same | Same |
| `index.html` | `no-cache` | Catch-all `e.GET("/*", ...)` | Read from embed FS, return as HTML |
| `/api/*` | None (N/A) | Echo group | Routed to handlers |
| `/health` | None (N/A) | Direct route | JSON status |

```go
//go:embed client/dist/*
var embedFS embed.FS

// In main():
subFS, _ := fs.Sub(embedFS, "client/dist")

// Hashed assets — immutable cache
e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/",
    http.FileServer(http.FS(subFS)))))

// SPA fallback — every non-API, non-asset route
e.GET("/*", func(c echo.Context) error {
    data, err := fs.ReadFile(subFS, "index.html")
    if err != nil {
        return c.NoContent(http.StatusNotFound)
    }
    c.Response().Header().Set("Cache-Control", "no-cache")
    return c.HTMLBlob(http.StatusOK, data)
})
```

**Dev mode** (`DEV_MODE=true`): static routes are NOT registered. Vite dev server handles frontend at `localhost:5173`. CORS allows that origin.

## 10. Build and Deployment Flow

```dockerfile
# Builder stage
FROM node:20-alpine AS frontend
WORKDIR /app/client
COPY client/package*.json ./
RUN npm ci
COPY client/ .
RUN npm run build   # → dist/

# Go build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY server/ .
COPY --from=frontend /app/client/dist ./client/dist
RUN go build -ldflags="-s -w" -o davechat ./cmd/main.go

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/davechat /usr/local/bin/davechat
EXPOSE 8080
CMD ["davechat"]
```

**Build sequence**:
1. `cd client && npm run build` → produces `client/dist/`
2. `cd server && go build -o davechat ./cmd/main.go` — embeds `client/dist/` via `//go:embed`
3. Binary is self-contained. Deploy as single Docker image

**Dev workflow:**
```bash
docker compose up -d mariadb          # DB only
cd client && npm run dev              # Vite on :5173
cd server && DEV_MODE=true go run ./cmd/main.go  # Echo on :8080
```

## Interfaces / Contracts

```go
// New middleware signature
func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // ... parse JWT from c.Request().Header ...
            c.Set("user_id", userID)
            return next(c)
        }
    }
}

// Standard handler signature across all handlers
type AuthHandler struct {
    DB     *sql.DB
    Config *config.Config
}
func (h *AuthHandler) Register(c echo.Context) error { ... }
func (h *AuthHandler) Login(c echo.Context) error { ... }
func (h *AuthHandler) Me(c echo.Context) error { ... }
func (h *AuthHandler) Refresh(c echo.Context) error { ... }
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Build | Compilation | `cd server && go build ./...` after each phase |
| Manual | All existing routes respond correctly | Start server, hit each endpoint with curl |
| Manual | Refresh token rotation | Login → use refresh twice → second call must fail |
| Manual | Search endpoint | Insert message with known term, search, verify result |
| Manual | Static serving | Build binary, `GET /` returns index.html, `GET /assets/index-xxxx.js` returns JS |

## Migration / Rollout

**Phase order**: Phase 1 (DB) → Phase 2 (Echo) → Phase 3 (Embed). Each phase is a separate commit.

**Phase 1**: SQL-only. Docker Compose restart recreates tables. No app changes needed.

**Phase 2**: Deployment window. Echo routes must be verified against all existing client API calls. The chi branch kept as `refs/changes/chi-router` for emergency rollback.

**Phase 3**: Requires client build artifact. The `VITE_API_URL` env var is no longer needed (same origin). Update `.env` and deployment config: remove `ALLOWED_ORIGINS`, add `DEV_MODE=false`.

## Open Questions

- [ ] Should `refresh_tokens` also store a device/IP for audit? Current scope: no, kept minimal. Can add later.
- [ ] Client-side: does the frontend need WebSocket path change from `/ws` to `/api/ws`? Current Chi mounts at `/ws`, Echo mounts at `/api/ws` under the protected group — client must update the WS connection URL.
