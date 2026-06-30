# Tasks: Echo Migration + DB Enhancements

## Review Workload Forecast

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900-1100 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: Phase 1 → PR 2: Phase 2 → PR 3: Phase 3 |
| Delivery strategy | (not specified — user must choose) |

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | DB schema + new Chi endpoints (additive, no migration) | PR 1 | base = main; standalone — works before any migration |
| 2 | Chi → Echo v4 migration, all handlers | PR 2 | base = main; depends on PR 1 for new handlers to also migrate |
| 3 | Single-binary embed + Dockerfile | PR 3 | base = main; depends on PR 2 for Echo static serving |

## Phase 1: DB Schema + New Endpoints (Additive)

- [ ] 1.1 `infra/mariadb/init.sql` — add `read_receipts`, `refresh_tokens`, `call_logs` tables, FULLTEXT index on `messages.content`
- [ ] 1.2 `DB.md` — document new tables (columns, FKs, indexes)
- [ ] 1.3 `server/internal/handlers/auth.go` — add `Refresh()` handler: token hash lookup, rotation (DELETE old + INSERT new), SHA-256 hashing
- [ ] 1.4 `server/internal/handlers/auth.go` — update `Login()` and `Register()` to also issue and persist refresh token
- [ ] 1.5 `server/internal/handlers/message.go` — add `Search()` handler: `GET /conversations/{id}/messages/search?q=`, FULLTEXT MATCH AGAINST BOOLEAN MODE
- [ ] 1.6 `server/internal/handlers/message.go` — add `MarkRead()` handler: `POST /conversations/{id}/read`, INSERT IGNORE into read_receipts
- [ ] 1.7 Create `server/internal/handlers/call_log.go` — `CallLogHandler` with `List()` (`GET /api/call-logs`) and `Create()` (`POST /api/call-logs`)
- [ ] 1.8 `server/cmd/main.go` — register new Chi routes: `/api/auth/refresh`, `/api/call-logs`, search, read
- [ ] 1.9 `client/src/lib/api.js` — add `refreshToken()`, `searchMessages()`, `markRead()`, `getCallLogs()`, `createCallLog()` to ApiClient

## Phase 2: Chi → Echo v4 Migration

- [x] 2.1 `server/go.mod` — remove `go-chi/chi/v5`, `go-chi/cors`; add `github.com/labstack/echo/v4`
- [ ] 2.2 `server/internal/config/config.go` — replace `AllowedOrigins` with per-env CORS logic, add `DevMode bool` field
- [x] 2.3 `server/internal/middleware/auth.go` — convert to `echo.MiddlewareFunc`, use `c.Set("user_id", id)` instead of `context.WithValue`
- [x] 2.4 `server/internal/handlers/auth.go` — convert `Register`, `Login`, `Me`, `Refresh` to `func(c echo.Context) error`, use `c.Bind()`, `c.Param()`, `c.JSON()`
- [x] 2.5 `server/internal/handlers/profile.go` — convert `List`, `Get`: replace `chi.URLParam` → `c.Param`, `context.Value` → `c.Get`
- [x] 2.6 `server/internal/handlers/conversation.go` — convert `List`, `Create`, `Get` to Echo pattern
- [x] 2.7 `server/internal/handlers/message.go` — convert `List`, `Create`, `Search`, `MarkRead` to Echo pattern
- [x] 2.8 `server/internal/handlers/call_log.go` — convert `List`, `Create` to Echo pattern
- [x] 2.9 `server/internal/handlers/helpers.go` — keep `errorResponse` type, remove `writeJSON`/`writeError` (handlers use `c.JSON` directly)
- [x] 2.10 `server/internal/websocket/handler.go` — convert `HandleWebSocket` to accept `(hub *Hub, c echo.Context)` error, extract userID from `c.Get("user_id")`, use `c.Response().Writer` for upgrade
- [x] 2.11 `server/cmd/main.go` — rewrite: `echo.New()`, custom `HTTPErrorHandler`, route groups (public/protected), CORS, Echo-style WS handler via closure

## Phase 3: Single Binary (Embed)

- [ ] 3.1 `client/vite.config.js` — add `base: '/'`, explicit `build.outDir: 'dist'`
- [ ] 3.2 Build client: `cd client && npm run build` produces `client/dist/`
- [ ] 3.3 `server/cmd/main.go` — add `//go:embed client/dist/*` var, `fs.Sub` for embedded FS
- [ ] 3.4 `server/cmd/main.go` — register static routes: `/assets/*` with immutable cache; SPA fallback `/*` serving `index.html` (guarded by `!cfg.DevMode`)
- [ ] 3.5 `server/internal/config/config.go` — load `DEV_MODE` env var, remove old `ALLOWED_ORIGINS`
- [ ] 3.6 Create `server/Dockerfile` — multi-stage: node:20 build dist, golang:1.25 build binary with embed, alpine:3.20 runtime
- [ ] 3.7 `infra/docker-compose.yml` — add `davechat-server` service building from `server/Dockerfile`, depends on mariadb
