# Proposal: Security Fixes v1

## Intent

Fix 4 audit vulnerabilities: JWT token leakage via URL params, missing WebSocket origin check, absent security headers, and user enumeration on registration.

## Scope

### In Scope
- Remove JWT query param fallback from auth middleware
- Configure Echo Logger to omit query strings
- Validate WebSocket `Origin` against allowed origins
- Pass JWT via WebSocket sub-protocol instead of URL
- Add security headers middleware (CSP, HSTS, X-Frame-Options, X-Content-Type-Options)
- Unify registration error messages to prevent user enumeration
- Update client `websocket.js` to send token via sub-protocol

### Out of Scope
- Rate limiting changes, input sanitization, XSS fixes
- Refresh token rotation overhaul
- Full auth rewrite or OAuth2 migration
- DB-level encryption at rest

## Capabilities

### New Capabilities
- `security-headers`: Security response headers (CSP, HSTS, XFO, XCTO, Referrer-Policy)

### Modified Capabilities
- `user-auth`: Registration endpoint returns single generic error for conflicts; auth middleware rejects query-param tokens
- `websocket-connection`: Auth moves from URL query param to sub-protocol header; Origin validation added

## Approach

1. **VULN-002 (auth middleware)**: Remove `c.QueryParam("token")` fallback. Update error message to reflect header-only auth. Configure Echo Logger format to exclude query strings.
2. **VULN-003 (websocket)**: Replace permissive `CheckOrigin` with host-based validation against `AllowedOrigins`. Extract JWT from `Sec-WebSocket-Protocol` during upgrade handshake, validate before `Upgrade()`. Update `websocket.js` to pass token as second arg to `WebSocket()` constructor (sets sub-protocol).
3. **VULN-004 (main.go)**: Add `echo.MiddlewareFunc` setting `Content-Security-Policy`, `Strict-Transport-Security`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy`, `Permissions-Policy`.
4. **VULN-005 (auth handler)**: Use one generic response for all registration conflicts — no distinct email vs username messages.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `server/internal/middleware/auth.go` | Modified | Remove query param token parsing |
| `server/internal/websocket/handler.go` | Modified | Origin validation + sub-protocol auth |
| `server/internal/handlers/auth.go` | Modified | Unified reg error messages |
| `server/cmd/main.go` | Modified | Security headers middleware + logger config |
| `client/src/lib/websocket.js` | Modified | Token via sub-protocol, not URL |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Sub-protocol unsupported | Low | All modern browsers support it |
| Existing WS sessions break during deploy | Med | Brief disconnect; auto-reconnect handles it |
| CSP blocks legitimate resources | Med | Start in Report-Only, tune before enforce |
| HSTS breaks local dev over HTTP | Low | Guard with DevMode flag |

## Rollback Plan

Revert all 5 file changes. No data migration involved — zero-downtime revert. For HSTS, set `max-age=0` on next deploy if already enforced.

## Dependencies

- None. All changes self-contained. No new external Go packages.

## Success Criteria

- [ ] Auth middleware returns 401 when `?token=` query param is used
- [ ] Disallowed WebSocket origins are rejected
- [ ] WebSocket connects successfully with sub-protocol token (no URL param)
- [ ] Registration returns only generic error on conflict
- [ ] All HTTP responses include CSP, HSTS, XFO, XCTO headers
- [ ] No JWT tokens appear in server access logs
- [ ] Login, messaging, calls work unchanged
