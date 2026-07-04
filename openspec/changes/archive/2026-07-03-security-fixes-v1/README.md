# Archive: security-fixes-v1

**Archived**: 2026-07-03
**Status**: Complete — all 9 tasks verified, PASS verdict, no CRITICAL issues.

## Change Summary

Fix 4 audit vulnerabilities:
- **VULN-002**: JWT token leakage via URL query params — removed `?token=` fallback from auth middleware; logger uses `${path_rfc3960}` to omit query strings
- **VULN-003**: Missing WebSocket origin check — origin validation + JWT via sub-protocol header (not URL)
- **VULN-004**: Absent security headers — 6 headers (CSP-RO, HSTS, XFO, XCTO, Referrer-Policy, Permissions-Policy) with DevMode gate
- **VULN-005**: User enumeration on registration — unified generic error for all conflict types

## Source of Truth Updated

| Domain | Action | Requirements |
|--------|--------|-------------|
| `openspec/specs/user-auth/spec.md` | Updated | Register (generic errors), JWT Middleware (header-only), Logger Omission (new) |
| `openspec/specs/websocket-connection/spec.md` | **Created** | Connection (sub-protocol auth), Origin Validation, Client Sub-Protocol Token |
| `openspec/specs/security-headers/spec.md` | **Created** | DevMode Gate, CSP, HSTS, XFO, XCTO, Referrer-Policy, Permissions-Policy |
| `openspec/specs/signaling/spec.md` | Updated | WebSocket Connection requirement moved to its own spec |

## Archive Contents

- `proposal.md` — change intent, scope, approach
- `specs/user-auth/spec.md` — delta spec for user-auth changes
- `specs/websocket-connection/spec.md` — delta spec for websocket connection changes
- `specs/security-headers/spec.md` — delta/full spec for new security headers capability
- `design.md` — architecture decisions and data flow
- `tasks.md` — 9 tasks, all complete
- `verify-report.md` — PASS, 10/10 tests, no issues

## SDD Cycle Complete

This change has been fully planned, implemented, verified, and archived.
