# Delta for Messages

## ADDED Requirements

### Requirement: Search Messages

(See full spec at `openspec/specs/message-search/spec.md` for search endpoint requirements and scenarios.)

### Requirement: Mark Messages as Read

The system MUST expose `POST /api/conversations/{id}/read`. The endpoint accepts an optional `message_ids` array. If omitted, marks ALL unread messages in the conversation as read.

#### Scenario: Mark specific messages as read

- GIVEN conversation C has messages M1 and M2 unread by Alice
- WHEN Alice calls `POST /api/conversations/C/read` with `{message_ids: ["M1"]}`
- THEN a `read_receipts` row is inserted for (M1, Alice)
- AND M2 remains unread

#### Scenario: Mark all as read

- GIVEN conversation C has 10 unread messages for Alice
- WHEN Alice calls `POST /api/conversations/C/read` with no body
- THEN `read_receipts` rows are inserted for all unread messages
- AND response includes `{marked: 10}`

#### Scenario: Idempotent re-read

- GIVEN Alice already read message M1
- WHEN Alice calls read again for M1
- THEN no duplicate row is inserted (PK constraint handles it)
- AND response is HTTP 200

### Requirement: Read Receipts Table

(See delta spec at `openspec/changes/echo-migration-and-db-enhancement/specs/database/spec.md` for `read_receipts` DDL.)

## MODIFIED Requirements

### Requirement: List Messages

(Previously: cursor-based pagination by `created_at`)

The system MUST expose `GET /api/conversations/{convID}/messages?cursor={cursor}&limit=50`. After Echo migration, URL param extraction changes to `c.Param("id")`.

#### Scenario: Returns messages with Echo handler signature

- GIVEN conversation C has 100 messages
- WHEN Alice calls `GET /api/conversations/C/messages?limit=50`
- THEN the handler uses `c.Param("id")` to get `convID`
- AND response is HTTP 200 with array of 50 messages

### Requirement: Send Message

(Previously: POST with chi URL params)

The system MUST expose `POST /api/conversations/{convID}/messages`. After Echo migration, handler signature changes to `func(c echo.Context) error`.

#### Scenario: Sender is participant sends message via Echo

- GIVEN Alice is a participant in conversation C
- WHEN Alice calls `POST /api/conversations/C/messages` with `{content: "Hola"}`
- THEN handler uses `c.Param("id")` and `c.Get("user_id")`
- AND response is HTTP 201
