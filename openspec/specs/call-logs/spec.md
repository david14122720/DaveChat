# Call Logs Specification

## Purpose

Persist call events (audio/video) from WebSocket signaling into `call_logs` and expose a query API.

## Requirements

### Requirement: Record Call

The system MUST insert a `call_logs` row when call lifecycle events occur via WebSocket.

#### Scenario: Completed call is persisted

- GIVEN Alice and Bob are connected via WebSocket
- WHEN `signal:offer` is sent, call proceeds, and `call:end` fires
- THEN a `call_logs` row is created with status `completed` and `duration_secs` computed
- AND a `messages` row of type `call` with `content` summarizing the call

#### Scenario: Missed call is recorded

- GIVEN Alice sends `signal:offer` to Bob who is offline
- WHEN `call:busy` is returned or timeout fires
- THEN a `call_logs` row is created with status `missed` and `ended_at` NULL

### Requirement: List Call History

The system MUST expose `GET /api/calls` returning the authenticated user's call history.

#### Scenario: Returns paginated call list

- GIVEN Alice has 15 past calls across conversations
- WHEN Alice calls `GET /api/calls?limit=10`
- THEN response is HTTP 200 with array of up to 10 calls ordered by `started_at DESC`
- AND each entry includes `{id, caller_id, callee_id, type, status, started_at, ended_at, duration_secs}`
- AND response includes `next_cursor` for pagination

#### Scenario: Empty history

- GIVEN Alice has no calls
- WHEN Alice calls `GET /api/calls`
- THEN response is HTTP 200 with empty array

### Requirement: Call Log Table Schema

The system MUST create `call_logs` with the following DDL:

```sql
CREATE TABLE IF NOT EXISTS call_logs (
    id CHAR(36) PRIMARY KEY,
    caller_id CHAR(36) NOT NULL,
    callee_id CHAR(36) NOT NULL,
    type ENUM('audio','video') NOT NULL,
    status ENUM('missed','completed','rejected','cancelled','busy') NOT NULL,
    started_at DATETIME(3) NOT NULL,
    ended_at DATETIME(3) NULL,
    duration_secs INT UNSIGNED DEFAULT 0,
    FOREIGN KEY (caller_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (callee_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX idx_call_logs_caller ON call_logs(caller_id);
CREATE INDEX idx_call_logs_callee ON call_logs(callee_id);
```

#### Scenario: Schema applies idempotently

- GIVEN `call_logs` table already exists
- WHEN the init SQL runs again
- THEN no error occurs (IF NOT EXISTS)
