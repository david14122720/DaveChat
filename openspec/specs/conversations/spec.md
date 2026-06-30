# Conversations Specification

## Purpose

Handlers for creating conversations, listing a user's conversations, and managing participants.

## Requirements

### Requirement: Create Direct Conversation

The system MUST expose `POST /api/conversations` accepting a `participant_ids` array (exactly 2 for direct chats).

#### Scenario: Creates conversation with two users

- GIVEN Alice and Bob are registered
- WHEN Alice calls `POST /api/conversations` with `{participant_ids: ["<bob-id>"]}`
- THEN response is HTTP 201 with `{id, created_at, participants: [...]}`
- AND both Alice and Bob are added to `participants`

#### Scenario: Existing conversation is reused

- GIVEN Alice and Bob already share a conversation
- WHEN Alice calls `POST /api/conversations` with Bob's ID
- THEN response is HTTP 200 with the existing conversation (idempotent)

### Requirement: List My Conversations

The system MUST expose `GET /api/conversations` returning conversations the authenticated user belongs to, ordered by `last_message_at DESC`.

#### Scenario: Returns conversations with last message preview

- GIVEN Alice belongs to two conversations with messages
- WHEN Alice calls `GET /api/conversations`
- THEN response is HTTP 200 with array ordered by `last_message_at DESC`
- AND each entry includes `{id, last_message: {content, sender_id, created_at}, participants: [...]}`

#### Scenario: Empty list

- GIVEN Alice belongs to no conversations
- WHEN Alice calls `GET /api/conversations`
- THEN response is HTTP 200 with empty array
