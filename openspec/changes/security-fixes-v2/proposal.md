# Proposal: Security Fixes v2

## Intent

Fix VULN-007 (per-account rate limiting on auth endpoints) and VULN-010 (uploads served without authentication).

## Scope

### In Scope
- VULN-007: Replace global rate limiter with per-IP rate limiting on `/api/auth/login` and `/api/auth/register`
- VULN-010: Protect `/uploads/` static file serving behind auth middleware

### Out of Scope
- Database-level changes, external rate limiting service, CDN integration

## Capabilities

### Modified Capabilities
- `user-auth`: Rate limiting — per-IP limits on login/register instead of global limit
- `user-auth`: Upload protection — avatar images require valid JWT to access

## Approach

### VULN-007 — Per-IP Rate Limiting
Replace the Echo global `middleware.RateLimiter(store)` with per-IP rate limiters on auth routes only. Use `golang.org/x/time/rate` to create per-IP limiters stored in a `sync.Map`, with periodic cleanup of stale entries.

### VULN-010 — Uploads Auth Protection
Remove `e.Static("/uploads", cfg.UploadDir)` from root Echo level. Create an auth-protected static file handler that validates JWT before serving files from the upload directory. Mount it on the `protected` group.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `server/cmd/main.go` | Modified | Rate limiter config + uploads route |
| `server/internal/middleware/ratelimit.go` | **New** | Per-IP rate limiter middleware for auth routes |
| `server/internal/middleware/static.go` | **New** | Auth-protected static file handler |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Per-IP limiter memory leak (stale IPs) | Low | Cleanup goroutine every 10 min |
| Rate limiting breaks legitimate users | Low | Generous limits (10 req/min per IP on auth) |

## Rollback Plan

Revert main.go changes to restore global rate limiter and unprotected uploads.

## Success Criteria

- [ ] Login/register from same IP >10 req/min → 429 Too Many Requests
- [ ] `/uploads/avatar.jpg` without auth → 401
- [ ] `/uploads/avatar.jpg` with valid JWT → serves file
- [ ] Other routes unaffected by rate limit changes
