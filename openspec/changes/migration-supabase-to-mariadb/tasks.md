# Tasks: Supabase → MariaDB Migration

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~950-1050 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 0 → PR 1 → PR 2 → PR 3 → PR 4 → PR 5 |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 0 | Foundation (DB, config, router) | PR 0 | Base branch: main. MariaDB schema, Go dependencies, config, chi router with health check |
| 1 | Auth (JWT, api.js, useAuth) | PR 1 | Depends on PR 0. JWT handlers, middleware, frontend fetch wrapper |
| 2 | Profiles + Conversations | PR 2 | Depends on PR 1. Handlers and MainLayout rewrite |
| 3 | Messages | PR 3 | Depends on PR 2. Handlers and ChatWindow rewrite |
| 4 | Signaling (WebSocket) | PR 4 | Depends on PR 0. Hub, client, WS handler |
| 5 | Cleanup | PR 5 | Depends on PR 4. Delete supabase/, remove deps, rewrite docs |

## Phase 0: Foundation

- [ ] 0.1 Create `infra/docker-compose.yml` — MariaDB 11 + Go server services with volumes
- [ ] 0.2 Create `db/init/01-schema.sql` — tables `users`, `conversations`, `participants`, `messages` with indexes
- [ ] 0.3 Create `server/internal/config/config.go` — env-based loader for DB DSN, JWT secret, port, allowed origins
- [ ] 0.4 Rewrite `server/internal/database/database.go` — MySQL driver with `go-sql-driver/mysql`, connection pool (5-20), ping
- [ ] 0.5 Rewrite `server/cmd/main.go` — chi router, CORS, logger, recoverer, `/health`, route groups
- [ ] 0.6 Rewrite `server/cmd/test_db/main.go` — MySQL ping instead of pgx query
- [ ] 0.7 Rewrite `server/go.mod` — remove pgx, add `go-sql-driver/mysql`, `go-chi/chi/v5`, `golang-jwt/jwt/v5`, `gorilla/websocket`, `bcrypt`
- [ ] 0.8 Update `server/.env` — replace DATABASE_URL with DB_HOST/PORT/USER/PASS/NAME, add JWT_SECRET
- [ ] 0.9 Update `client/.env` — add VITE_API_URL=http://localhost:8080

## Phase 1: Auth

- [ ] 1.1 Create `server/internal/middleware/auth.go` — JWT extraction from Bearer header, validation, inject user_id into context
- [ ] 1.2 Create `server/internal/handlers/auth.go` — register (bcrypt + INSERT), login (SELECT + compare + JWT sign), me
- [ ] 1.3 Create `client/src/lib/api.js` — fetch wrapper with JWT from localStorage, auto-logout on 401
- [ ] 1.4 Rewrite `client/src/hooks/useAuth.js` — replace supabase.auth calls with api.js register/login/logout, store JWT in localStorage

## Phase 2: Profiles + Conversations

- [ ] 2.1 Create `server/internal/handlers/profile.go` — list all (exclude self), get by ID, update own (username/avatar)
- [ ] 2.2 Create `server/internal/handlers/conversation.go` — create (idempotent direct), list with last_message, get by ID
- [ ] 2.3 Rewrite `client/src/components/MainLayout.jsx` — fetch profiles via api.js, remove supabase import

## Phase 3: Messages

- [ ] 3.1 Create `server/internal/handlers/message.go` — list with cursor pagination, send (verify participant)
- [ ] 3.2 Rewrite `client/src/components/ChatWindow.jsx` — fetch/send messages via api.js with 2s polling, remove supabase import

## Phase 4: Signaling

- [x] 4.1 Create `server/internal/websocket/hub.go` — connection registry, room per user, relay to target, presence broadcast
- [x] 4.2 Create `server/internal/websocket/handler.go` — read/write pumps with gorilla/websocket, 30s ping/pong heartbeat, read deadline, message dispatch
- [x] 4.3 Create `server/internal/websocket/handler.go` — WS upgrade with JWT context, dispatch offer/answer/ICE/call events
- [x] 4.4 Update `server/cmd/main.go` — add `GET /ws` route to chi router

## Phase 5: Cleanup

- [ ] 5.1 Delete `supabase/` directory (migrations, seed)
- [ ] 5.2 Delete `client/src/lib/supabaseClient.js`
- [ ] 5.3 Remove `@supabase/supabase-js` from `client/package.json`
- [ ] 5.4 Rewrite `DB.md` — MariaDB schema without RLS/triggers
- [ ] 5.5 Rewrite `RESUMEN.md` — new architecture (Go API + MariaDB, no Supabase)
