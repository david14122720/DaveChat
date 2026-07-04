# Tasks: Security Fixes v1

## Review Workload Forecast

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | All 4 fixes | PR 1 | ~160 lines, single deploy; client change backward-compatible |

## Phase 1: VULN-005 — User Enumeration Fix

- [x] **T-001**: Unify registration errors in `server/internal/handlers/auth.go` (lines 118-131). Replace both distinct conflict responses (DUPLICATE_EMAIL, DUPLICATE_USERNAME) with a single generic `409 {"error":"Registration failed. Email or username may already be in use."}`. **Files**: `server/internal/handlers/auth.go`. **Dep**: none. **Verify**: duplicate email and duplicate username both return the same generic message. **Est**: Small (~10 lines).

## Phase 2: VULN-002 — Auth Middleware & Logger

- [x] **T-002**: Remove `?token=` query param fallback in `server/internal/middleware/auth.go` (delete lines 33-35). Update error message on line 39 from `"Missing Authorization header or token query param"` to `"Authorization header required"`. **Files**: `server/internal/middleware/auth.go`. **Dep**: none. **Verify**: request with `?token=valid_jwt` and no `Authorization` header returns 401 with "Authorization header required". **Est**: Small (~5 lines).

- [x] **T-003**: Replace `e.Use(middleware.Logger())` in `server/cmd/main.go` with `middleware.LoggerWithConfig(middleware.LoggerConfig{Format: "${time_rfc3339} ${method} ${path_rfc3960} ${status} ${latency_human} ${bytes_in} ${bytes_out}\n"})`. **Files**: `server/cmd/main.go`. **Dep**: none. **Verify**: request to `/api/auth/login?token=secret` appears in logs as `/api/auth/login` without query string. **Est**: Small (~5 lines).

## Phase 3: VULN-004 — Security Headers

- [x] **T-004**: Create `server/internal/middleware/headers.go` with `SecurityHeadersMiddleware(devMode bool) echo.MiddlewareFunc`. Headers: `Content-Security-Policy-Report-Only`, `Strict-Transport-Security: max-age=31536000; includeSubDomains`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` (camera/mic/geolocation restricted). When `devMode=true`, skip all headers. **Files**: `server/internal/middleware/headers.go` (create). **Dep**: none. **Verify**: production responses include all 6 headers; dev responses include none. **Est**: Medium (~60 lines).

- [x] **T-005**: Register `middleware.SecurityHeadersMiddleware(!cfg.DevMode)` in `server/cmd/main.go` after the existing `e.Use(middleware.Recover())`. **Files**: `server/cmd/main.go`. **Dep**: T-004. **Verify**: headers present with `DEV_MODE=false`, absent with `DEV_MODE=true`. **Est**: Small (~3 lines).

## Phase 4: VULN-003 — WebSocket Sub-Protocol Auth

- [x] **T-006**: Refactor `HandleWebSocket` in `server/internal/websocket/handler.go` from `HandleWebSocket(hub *Hub, c echo.Context)` to factory `HandleWebSocket(hub *Hub, jwtSecret string, allowedOrigins []string) echo.HandlerFunc`. The returned handler: extracts JWT from `Sec-WebSocket-Protocol` header, parses+validates it with `jwt.Parse`, validates `Origin` against allowed origins (non-empty slice) or falls back to Host comparison, echoes sub-protocol back via `Upgrade()` response header. Replace global `var upgrader` with local instance in factory with `CheckOrigin` closure. `ReadBufferSize`/`WriteBufferSize` remain 1024. **Files**: `server/internal/websocket/handler.go`. **Dep**: none. **Verify**: valid sub-protocol JWT + allowed origin → 101 upgrade; missing/expired JWT → 403; disallowed origin → 403. **Est**: Medium (~70 lines).

- [x] **T-007**: Move `/ws` route from `protected` group to `api` group in `server/cmd/main.go`. Change from `protected.GET("/ws", func(c echo.Context) error { return ws.HandleWebSocket(hub, c) })` to `api.GET("/ws", ws.HandleWebSocket(hub, cfg.JWTSecret, strings.Split(cfg.AllowedOrigins, ",")))`. Remove `/ws` from protected group. Add `"strings"` to imports. **Files**: `server/cmd/main.go`. **Dep**: T-006. **Verify**: WS connects on `/api/ws` via sub-protocol JWT, no auth middleware needed. **Est**: Small (~5 lines).

- [x] **T-008**: Update `client/src/lib/websocket.js`. Change URL construction from `` `${protocol}://${host}/api/ws?token=${token}` `` to `` `${protocol}://${host}/api/ws` `` and `new WebSocket(url)` to `new WebSocket(url, [token])`. **Files**: `client/src/lib/websocket.js`. **Dep**: T-007 (logical order only; deploy must coordinate). **Verify**: browser devtools shows `Sec-WebSocket-Protocol` header with JWT; `/api/ws` URL has no query params. **Est**: Small (~2 lines).

## Phase 5: Testing

- [x] **T-009**: Unit tests for auth middleware (request with `?token=` → 401), security headers middleware (headers present when devMode=false, absent when true), and WS origin validation (allowed origin allows, disallowed origin rejects, missing origin rejects). **Files**: create tests alongside each component. **Dep**: T-002, T-004, T-006. **Verify**: all tests pass. **Est**: Medium (~60 lines).
