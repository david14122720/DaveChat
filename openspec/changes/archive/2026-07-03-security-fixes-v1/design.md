# Design: Security Fixes v1

## Technical Approach

Four independent security fixes targeting JWT leakage (VULN-002, VULN-003), missing HTTP security headers (VULN-004), and user enumeration (VULN-005). Each fix touches a different file with no cross-cutting concerns. The WebSocket handler is the only structural change: it becomes a self-contained handler factory that owns its auth and origin validation, moving out of the shared auth middleware.

## Architecture Decisions

### Decision: Auth Middleware — Query param removal

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Keep query param fallback | Convenient for dev, leaks tokens in logs | ❌ Reject |
| Remove lines 33–35 only | Minimal diff, error message still mentions query param | ✅ Remove + update error text |

**Rationale**: Strip lines 33–35 (`c.QueryParam("token")`). Change error on line 39 to `"Authorization header required"`. Token leakage is the vulnerability — no half measures.

### Decision: Echo Logger — Query string omission

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Custom logger middleware | Full control, more code to maintain | ❌ Not needed |
| Custom format via `LoggerConfig` | Replaces `${uri}` with `${path_rfc3960}` — query strings never logged | ✅ Zero new code |

**Rationale**: Echo's `LoggerConfig.Format` token `${path_rfc3960}` emits the raw URL path without query string. Replace the default format. No custom logger needed.

### Decision: WebSocket Origin validation

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Compare to Host header | Works in dev, weaker security | ❌ |
| Config `AllowedOrigins` list | Explicit, secure, needs config | ✅ Primary approach |
| Fallback: Host comparison if list empty | Graceful for unconfigured setups | ✅ Fallback |

**Rationale**: Split `AllowedOrigins` by comma into a slice (currently single string). If slice is non-empty, validate against it; otherwise fall back to `Host` header comparison. `CheckOrigin` becomes a closure capturing the slice.

### Decision: WebSocket sub-protocol auth — handler factory pattern

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Keep WS in `protected` group, modify `AuthMiddleware` to handle sub-protocol | Middleware becomes WS-aware, mixed concerns | ❌ |
| Remove WS from `protected` group, move auth into `HandleWebSocket` | Handler is self-contained, owns its auth | ✅ Factory pattern |

**Rationale**: `HandleWebSocket` becomes `HandleWebSocket(hub, jwtSecret, allowedOrigins) echo.HandlerFunc`. It extracts the JWT from `Sec-WebSocket-Protocol` header, validates it, then calls `upgrader.Upgrade()` echoing back the sub-protocol in response headers. Route becomes `api.GET("/ws", ws.HandleWebSocket(...))` — no longer behind `protected`.

### Decision: Security Headers — Custom vs `middleware.Secure()`

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `middleware.Secure()` | No CSP, no Referrer-Policy, no Permissions-Policy | ❌ Insufficient |
| Custom middleware | Full control, 6 headers, DevMode gate | ✅ |

**Rationale**: Build a custom `SecurityHeadersMiddleware(devMode bool)` in `server/internal/middleware/headers.go`. CSP starts in Report-Only mode. DevMode guard per spec.

## Data Flow

### WebSocket Upgrade flow (VULN-003)

```
Client                    Server
  │                         │
  │──── WS /api/ws ────────→│  (with Sec-WebSocket-Protocol: <jwt>)
  │                         │
  │    HandleWebSocket      │
  │    ┌────────────────┐   │
  │    │ 1. Extract      │   │
  │    │    Sec-WebSocket│   │
  │    │    -Protocol    │   │
  │    │ 2. Parse+validate│   │
  │    │    JWT          │   │
  │    │ 3. Check Origin │   │
  │    │ 4. Upgrade()    │   │
  │    │    (echo sub-   │   │
  │    │     protocol)   │   │
  │    └────────────────┘   │
  │←── 101 Switching ───────│
  │    Protocols: <jwt>     │
```

### Auth validation flow (VULN-002)

```
Request ──→ Echo Logger ──→ AuthMiddleware ──→ Handler
                │                │
                │ ${path_rfc3960}│ Only checks
                │ (no ?token=)   │ Authorization: Bearer
                │                │ ?token= → ignored → 401
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `server/internal/middleware/auth.go` | Modify | Remove query param fallback (lines 33-35), update error message |
| `server/internal/middleware/headers.go` | **Create** | Security headers middleware (CSP, HSTS, XFO, XCTO, Referrer-Policy, Permissions-Policy) |
| `server/internal/websocket/handler.go` | Modify | Factory pattern: sub-protocol JWT auth + origin validation |
| `server/internal/handlers/auth.go` | Modify | Unify registration conflict errors (lines 118-131) |
| `server/cmd/main.go` | Modify | Replace default logger with custom format; register security headers middleware; update WS route |
| `server/internal/config/config.go` | Modify | Minor: unused if `AllowedOrigins` split happens inside handler (no config change needed, split at use-site) |
| `client/src/lib/websocket.js` | Modify | Change `new WebSocket(url)` to `new WebSocket(url, [token])` — removes `?token=` from URL |

## Interfaces / Contracts

```go
// New factory signature in handler.go
func HandleWebSocket(hub *Hub, jwtSecret string, allowedOrigins []string) echo.HandlerFunc

// New middleware in headers.go — used in main.go
func SecurityHeadersMiddleware(devMode bool) echo.MiddlewareFunc
```

Logger format change in `main.go`:
```go
e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
    Format: "${time_rfc3339} ${method} ${path_rfc3960} ${status} ${latency_human} ${bytes_in} ${bytes_out}\n",
}))
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Auth middleware | Request with `?token=` → expect 401 + header-only error text |
| Unit | Security headers | Handler returns headers when devMode=false, none when devMode=true |
| Unit | WS origin validation | Upgrader.CheckOrigin with allowed/disallowed/missing origins |
| Manual | WS sub-protocol | `new WebSocket(url, [token])` → handshake succeeds; missing token → 403 |
| Manual | Logger | Start server, hit `/api/auth/login?token=foo` → log shows `/api/auth/login` only |
| Manual | Registration | Duplicate email/username → generic error, no distinction |

## Migration / Rollout

1. **Client first** (backward-compatible): deploy `websocket.js` change. Old clients still connect via `?token=` until server is deployed.
2. **Server deployment** (breaking moment): deploy all server changes together. Old clients with `?token=` will get 401 on WS — their reconnect logic triggers the new sub-protocol path.
3. **No staggered server deploys**: all 4 server changes must go together to avoid leaving VULN-002/VULN-003 open during rollout.
4. **HSTS note**: once deployed in production, HSTS is cached by browsers. Rollback must set `max-age=0`.

## Open Questions

None.
