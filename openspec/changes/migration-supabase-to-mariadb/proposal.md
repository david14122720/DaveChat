# Proposal: Migration Supabase → MariaDB + Go API Server

## Intent

Remove ALL Supabase dependencies (auth, DB, realtime) from DaveChat. Replace with MariaDB via Docker, a full Go API server handling auth/profiles/messages/signaling, and a React frontend talking exclusively to the Go API. This eliminates third-party cloud lock-in, reduces operational cost, and gives full control over the data model and auth flow.

## Scope

### In Scope
- MariaDB schema (users, conversations, messages, profiles) + Go SQL driver
- JWT auth (register/login handlers, middleware)
- Go REST API: profiles, conversations, messages
- Go WebSocket signaling hub
- React frontend: swap Supabase calls for Go API calls
- docker-compose.yml with MariaDB + Go server
- Delete all `supabase/` files and `@supabase/supabase-js` dependency

### Out of Scope
- Tests (no test infrastructure exists — deferred)
- TypeScript migration
- CI/CD setup
- Password reset flow (first version)
- File/image upload (future phase)

## Capabilities

### New Capabilities
- `database`: MariaDB schema (migrations, users/conversations/messages tables)
- `user-auth`: JWT registration, login, middleware (replaces Supabase Auth)
- `user-profiles`: CRUD profile handlers
- `conversations`: Conversation creation, listing, membership
- `messages`: Send, list, paginate messages per conversation
- `signaling`: WebSocket hub for real-time events (calls, presence)
- `api-gateway`: Go HTTP router exposing REST + WebSocket on one port

### Modified Capabilities
None — no existing specs to modify.

## Approach

Six phased vertical slices, each independently shippable:

| Phase | Deliverable | Changes |
|-------|-------------|---------|
| 0 — Foundation | docker-compose.yml, MariaDB schema, pgx→mysql driver, config loader | 4 files |
| 1 — Auth | JWT middleware, register/login handlers, api.js client, useAuth.js rewrite | 6 files |
| 2 — Profiles+Convs | Handlers and MainLayout.jsx rewrite (uses API, not Supabase) | 5 files |
| 3 — Messages | Message handler + ChatWindow.jsx rewrite | 3 files |
| 4 — Signaling | WebSocket hub + client + signaling handler | 5 files |
| 5 — Cleanup | Delete supabase/, uninstall @supabase/supabase-js, update docs | 6 files |

Each phase is fully functional — no regression between phases.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `server/internal/database/` | Rewrite | pgx→MySQL driver, new schema |
| `server/internal/auth/` | New | JWT middleware + handlers |
| `server/cmd/server/main.go` | Rewrite | Router setup, handlers init |
| `server/go.mod` | Update | Add gorilla/mux, jwt, mysql driver |
| `client/src/supabaseClient.js` | Delete | No longer needed |
| `client/src/hooks/useAuth.js` | Rewrite | Call Go API instead of Supabase |
| `client/src/components/ChatWindow.jsx` | Update | Messages via Go API |
| `client/src/components/MainLayout.jsx` | Update | Conversations via Go API |
| `client/src/api.js` | New | Axios/fetch wrapper for Go API |
| `client/package.json` | Update | Remove @supabase/supabase-js |
| `docker-compose.yml` | New | MariaDB + Go server |
| `.env.example` | Update | New env vars (DB, JWT secret) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Auth migration breaks existing sessions | High | JWT tokens stateless — users re-login once |
| Schema mismatch (PG types → MariaDB) | Medium | Use compatible types (VARCHAR/TEXT/JSON) |
| No tests = fragile refactors | High | Manual verification per phase, build guard |
| WebSocket signaling complexity | Low | Reuse existing `gorilla/websocket` patterns |

## Rollback Plan

Per-phase rollback: each phase has a git commit. Revert the commit to go back. For auth: old Supabase client code remains on the `supabase` branch — switch branch to restore. For DB: MariaDB container replaces Supabase entirely — no in-flight data migration (new schema, fresh start).

## Dependencies

- Docker + Docker Compose (dev)
- MariaDB:11 image
- Go libraries: `github.com/gorilla/mux`, `github.com/go-sql-driver/mysql`, `github.com/golang-jwt/jwt/v5`

## Success Criteria

- [ ] `docker compose up` starts MariaDB + Go API server
- [ ] User can register, login, and receive a JWT
- [ ] User can create conversations and send messages
- [ ] WebSocket signaling connects and relays events
- [ ] All Supabase imports removed; project builds without `@supabase/supabase-js`
- [ ] `go build ./...` and `cd client && npm run build` pass
