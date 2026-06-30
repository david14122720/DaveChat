# Messages Specification

## Purpose

Handlers for sending and listing messages per conversation with pagination.

## Requirements

### Requirement: Send Message

The system MUST expose `POST /api/conversations/{convID}/messages` accepting `content` and optional `type` (default `text`).

#### Scenario: Sender is participant sends message

- GIVEN Alice is a participant in conversation C
- WHEN Alice calls `POST /api/conversations/C/messages` with `{content: "Hola"}`
- THEN response is HTTP 201 with `{id, conversation_id, sender_id, content, type, created_at}`
- AND the message is stored in `messages`

#### Scenario: Non-participant cannot send

- GIVEN Alice is NOT a participant in conversation C
- WHEN Alice tries to send a message to C
- THEN response is HTTP 403

### Requirement: List Messages

The system MUST expose `GET /api/conversations/{convID}/messages?before=<id>&limit=50`.

#### Scenario: Returns messages in chronological order

- GIVEN conversation C has 100 messages
- WHEN Alice calls `GET /api/conversations/C/messages?limit=50`
- THEN response is HTTP 200 with array of 50 messages (oldest first)
- AND response includes `has_more: true` and `next_cursor`

#### Scenario: Empty conversation

- GIVEN conversation C has no messages
- WHEN Alice calls `GET /api/conversations/C/messages`
- THEN response is HTTP 200 with empty array
