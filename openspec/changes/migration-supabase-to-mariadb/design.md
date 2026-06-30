# Design: Migration Supabase → MariaDB + Go API Server

## Technical Approach

Six-phase vertical migration replacing Supabase (auth, DB, realtime) with a self-hosted stack: MariaDB 11 in Docker + Go API server (chi router, go-sql-driver/mysql, golang-jwt, gorilla/websocket). Each phase is independently shippable with no regression. The React frontend swaps `@supabase/supabase-js` calls for a thin `api.js` fetch wrapper targeting `localhost:8080`. Existing signaling WebSocket infrastructure (empty `internal/websocket/`, `internal/signaling/`) gets populated in Phase 4.

## Architecture Decisions

| Option | Tradeoff | Decision |
|--------|----------|----------|
| chi vs gorilla/mux | chi has middleware chaining, context helpers; mux is simpler | **chi** — cleaner middleware composition for auth/JWT |
| Polling vs WebSocket for messages | WS is realtime but adds complexity; polling is simpler for MVP | **Polling every 2s** — Phase 3 is simple; WebSocket reserved for signaling (Phase 4) |
| UUID v4 vs auto-increment | UUIDs safer for distributed + client-side generation | **UUID v4** — matches existing Supabase schema pattern |
| CHAR(36) vs BINARY(16) for UUIDs | BINARY(16) saves space but needs hex conversion | **CHAR(36)** — simpler, debuggable, matches existing code |
| Single Go binary vs separate services | One binary = simpler deploy | **Single binary** — chi routes both REST and WS on port 8080 |

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    Docker Compose                    │
│                                                      │
│  ┌──────────────────┐       ┌──────────────────┐    │
│  │   Go API Server  │       │    MariaDB 11     │    │
│  │   :8080          │◄─────►│   :3306           │    │
│  │                  │       │   volume: db_data │    │
│  │  ┌────────────┐  │       └──────────────────┘    │
│  │  │ chi router │  │                                │
│  │  │            │  │       ┌──────────────────┐    │
│  │  │ /api/auth  │  │       │    React Client   │    │
│  │  │ /api/prof. │  │◄──────┤   Vite :5173      │    │
│  │  │ /api/conv. │  │       │   api.js fetch    │    │
│  │  │ /api/msg   │  │       └──────────────────┘    │
│  │  │ /ws        │  │                                │
│  │  └────────────┘  │                                │
│  └──────────────────┘                                │
└─────────────────────────────────────────────────────┘
```

## Data Flow

### Auth Flow (Phase 1)
```
Client                    Go Server                   MariaDB
  │   POST /api/auth/register │                         │
  │   {email,password,username}│                         │
  │──────────────────────────►│                         │
  │                           │  bcrypt(password)       │
  │                           │  INSERT INTO profiles   │
  │                           │────────────────────────►│
  │                           │  ←── row (id, username) │
  │                           │  sign JWT (id, email)   │
  │  ←── {token, user}        │                         │
  │                           │                         │
  │   GET /api/auth/me        │                         │
  │   Authorization: Bearer JWT│                        │
  │──────────────────────────►│                         │
  │                           │  validate JWT, extract  │
  │                           │  user_id from context   │
  │  ←── {id, username, email}│                         │
```

### Message Flow (Phase 3)
```
Client A              Go Server              MariaDB           Client B
  │  POST /conv/{id}/msg │                      │                  │
  │  {content,type}      │                      │                  │
  │─────────────────────►│                      │                  │
  │                      │  INSERT INTO messages │                  │
  │                      │─────────────────────►│                  │
  │  ←── {id, created_at}│                      │                  │
  │                      │                      │                  │
  │                      │  ──[no push]──       │                  │
  │                      │                      │                  │
  │  ──[every 2s]──      │                      │                  │
  │  GET /conv/{id}/msg  │                      │                  │
  │  ?cursor=2024-01-01  │                      │                  │
  │─────────────────────►│  SELECT ... WHERE     │                  │
  │                      │  created_at > cursor  │                  │
  │                      │─────────────────────►│                  │
  │  ←── {messages[]}   │                      │                  │
```

### Signaling Flow (Phase 4)
```
Caller A               Go WS Hub               Callee B
  │  CONNECT /ws?token=JWT  │                      │
  │─────────────────────────►                      │
  │                         │  CONNECT /ws?token=JWT│
  │                         │◄─────────────────────│
  │  {type:"signal:offer",  │                      │
  │   sdp:...,target:B}     │                      │
  │─────────────────────────►                      │
  │                         │  {type:"signal:offer",│
  │                         │   sdp:...,from:A}    │
  │                         │─────────────────────►│
  │                         │                      │
  │                         │  {type:"signal:answer",│
  │                         │   sdp:...,from:B}    │
  │                         │◄─────────────────────│
  │  {type:"signal:answer", │                      │
  │   sdp:...,from:B}       │                      │
  │◄────────────────────────│                      │
  │                         │                      │
  │  ←── ping/pong 30s ──► │ ── ping/pong 30s ──►│
```

## Component Design Per Phase

### Phase 0 — Foundation (4 files created)

```
server/
├── Dockerfile                           # Multi-stage Go build
├── cmd/server/main.go                   # Rewrite: chi router, middleware, handlers
├── internal/
│   ├── config/config.go                 # NEW: env-based loader
│   └── database/database.go             # REWRITE: go-sql-driver/mysql
docker-compose.yml                       # NEW: MariaDB + Go server
db/init/01-schema.sql                    # NEW: MariaDB schema
```

### Phase 1 — Auth (6 files)

```
server/internal/
├── auth/
│   ├── handler.go                       # NEW: register, login, me handlers
│   └── middleware.go                    # NEW: JWT middleware
client/src/
├── api.js                               # NEW: fetch wrapper
└── hooks/useAuth.js                     # REWRITE: Go API instead of Supabase
```

### Phase 2 — Profiles + Conversations (5 files)

```
server/internal/
├── profiles/
│   └── handler.go                       # NEW: CRUD profiles
└── conversations/
    └── handler.go                       # NEW: CRUD conversations
client/src/components/
└── MainLayout.jsx                       # REWRITE: Go API instead of Supabase
```

### Phase 3 — Messages (3 files)

```
server/internal/
└── messages/
    └── handler.go                       # NEW: send/list messages
client/src/components/
└── ChatWindow.jsx                       # REWRITE: Go API + polling
```

### Phase 4 — Signaling (5 files)

```
server/internal/
├── signaling/hub.go                     # NEW: room management, message relay
└── websocket/
    ├── handler.go                       # NEW: WS upgrade, auth, heartbeat
    └── client.go                        # NEW: per-connection read/write pumps
client/src/
├── hooks/useWebRTC.js                   # MODIFY: WS endpoint URL
└── services/signaling.js                # NEW: WS client wrapper
```

### Phase 5 — Cleanup (6 files deleted/updated)

```
client/
├── package.json                         # MODIFY: remove @supabase/supabase-js
├── src/lib/supabaseClient.js            # DELETE
├── .env                                 # UPDATE: remove Supabase vars, add VITE_API_URL
supabase/                                # DELETE (entire directory)
server/.env                              # UPDATE: remove Supabase vars
DB.md                                    # REWRITE: MariaDB schema
RESUMEN.md                               # UPDATE: new architecture
```

## API Contract

### Auth

| Method | Path | Request | Response | Auth |
|--------|------|---------|----------|------|
| POST | `/api/auth/register` | `{username, email, password}` | `{token, user: {id, username, email}}` | No |
| POST | `/api/auth/login` | `{email, password}` | `{token, user: {id, username, email, avatar_url, status}}` | No |
| GET | `/api/auth/me` | — | `{id, username, email, avatar_url, status, last_seen, created_at}` | JWT |

### Profiles

| Method | Path | Request | Response | Auth |
|--------|------|---------|----------|------|
| GET | `/api/profiles` | — | `{profiles: [{id, username, avatar_url, status}]}` | JWT |
| GET | `/api/profiles/{id}` | — | `{id, username, avatar_url, status, last_seen}` | JWT |
| PUT | `/api/profiles/{id}` | `{username?, avatar_url?}` | `{id, username, avatar_url, updated_at}` | JWT (own) |

### Conversations

| Method | Path | Request | Response | Auth |
|--------|------|---------|----------|------|
| POST | `/api/conversations` | `{type:"direct"\|"group", participant_ids:[]}` | `{id, type, created_at}` | JWT |
| GET | `/api/conversations` | — | `{conversations: [{id, type, last_message, participants[], last_message_at}]}` | JWT |
| GET | `/api/conversations/{id}` | — | `{id, type, created_at, participants[]}` | JWT (member) |
| PUT | `/api/conversations/{id}` | `{type?}` | `{id, type, updated_at}` | JWT (member) |

### Messages

| Method | Path | Request | Response | Auth |
|--------|------|---------|----------|------|
| GET | `/api/conversations/{id}/messages` | `?cursor=ISO8601&limit=50` | `{messages: [{id, sender_id, content, type, created_at}], next_cursor}` | JWT (member) |
| POST | `/api/conversations/{id}/messages` | `{content, type:"text"\|"call"}` | `{id, conversation_id, sender_id, content, type, created_at}` | JWT (member) |

### WebSocket

| Endpoint | Query | Protocol |
|----------|-------|----------|
| `GET /ws` | `token=JWT` | gorilla/websocket, JSON messages |

**Signaling message envelope:**
```json
{
  "type": "signal:offer|signal:answer|signal:ice-candidate|call:reject|call:busy|call:end",
  "target": "uuid-of-recipient",
  "from": "uuid-of-sender",
  "sdp": "{SDP string}" | null,
  "candidate": "{ICE candidate}" | null
}
```

**Server heartbeat:**
```json
{"type": "ping"}
→ client responds: {"type": "pong"}
```

**Error envelope (all endpoints):**
```json
{
  "error": {
    "code": "UNAUTHORIZED|NOT_FOUND|VALIDATION_ERROR|INTERNAL_ERROR",
    "message": "Human-readable description"
  }
}
```

## HTTP Error Codes

| HTTP | Code | When |
|------|------|------|
| 400 | `VALIDATION_ERROR` | Missing fields, bad email, short username |
| 401 | `UNAUTHORIZED` | Missing/invalid/expired JWT |
| 403 | `FORBIDDEN` | JWT valid but not owner of resource |
| 404 | `NOT_FOUND` | Profile/conversation not found |
| 409 | `CONFLICT` | Duplicate email or username |
| 500 | `INTERNAL_ERROR` | DB failure, unexpected server error |

## Database Design (MariaDB)

### Schema

```
profiles (id, username, email, password_hash, avatar_url, status enum, last_seen, created_at, updated_at)
    │ 1
    │
    ├────────────────────────────────────────────┐
    │                                            │
    ▼                                            ▼
participants (conversation_id, user_id)  messages (id, conversation_id, sender_id, content, type, created_at)
    │                                            │
    │ FK: conversation_id                        │ FK: conversation_id
    ▼                                            ▼
conversations (id, type enum, created_at, last_message_at)
```

### Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| PRIMARY | `profiles(id)` | PK lookup |
| UNIQUE | `profiles(username)` | Username uniqueness + search |
| UNIQUE | `profiles(email)` | Login lookup |
| PRIMARY | `conversations(id)` | PK lookup |
| INDEX | `conversations(last_message_at DESC)` | Conversation list sort |
| PRIMARY | `participants(conversation_id, user_id)` | Compound PK |
| INDEX | `participants(user_id)` | "My conversations" query |
| PRIMARY | `messages(id)` | PK lookup |
| INDEX | `messages(conversation_id, created_at)` | Cursor pagination |

## Security Design

### Password Flow
```
Registration:  password → bcrypt(gensalt(12)) → password_hash → store
Login:         password → bcrypt.Compare(password_hash) → match → issue JWT
```

### JWT Lifecycle
```
Signing:     HS256, claims: {sub: user_id, email, iat, exp}
Expiration:  7 days (no refresh token in v1 — re-login required)
Validation:  ParseWithClaims → verify signature → check exp → extract sub
Context:     middleware injects "user_id" into chi context via context.WithValue
```

### Middleware Chain
```
chi router
├── CORS middleware (ALLOWED_ORIGINS)
├── Logger middleware
├── Recoverer middleware
│
├── /api/auth/*  (no JWT)
│
├── /api/*       (JWT middleware)
│   ├── /profiles
│   ├── /conversations
│   └── /conversations/{id}/messages
│
└── /ws          (JWT query param token validation)
```

### Error Handling Strategy

```
Handler → domain/business-logic error → HTTPError struct
                                           ├── code (machine-readable)
                                           ├── message (human-readable)
                                           ├── status (HTTP status)
                                           └── err (original, logged but not sent)

chi router catches panics via Recoverer middleware.
All errors formatted as: {"error": {"code": "...", "message": "..."}}
```

**Error categories:**
- **Validation**: 400, field-level messages in development, generic in production
- **Auth**: 401/403, never reveal if user exists (login: always "invalid credentials")
- **Not found**: 404, same response for missing conv vs. no permission
- **Internal**: 500, logged with stack trace, client gets generic "internal error"

## Migration / Rollout

No data migration from Supabase to MariaDB — fresh schema, new DB, users re-register. Each phase is a git commit. Rollback = `git revert <phase-commit>`. The `supabase` branch preserves the old architecture for reference.

## Open Questions

- [ ] Move `vite.config.js` proxy to avoid CORS issues during dev, or rely on CORS middleware in Go?
- [ ] Should Phase 4 WebSocket use the same port as REST (8080) via chi, or a separate port?
