# Delta for Database

## ADDED Requirements

### Requirement: Read Receipts Table

The system MUST create a `read_receipts` table that tracks per-message read status.

```sql
CREATE TABLE IF NOT EXISTS read_receipts (
    message_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    read_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (message_id, user_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
);
```

#### Scenario: Insert on first read

- GIVEN Alice opens a conversation with unread messages
- WHEN `POST /api/conversations/{id}/read` is called
- THEN `read_receipts` rows are inserted for each message
- AND duplicate inserts are ignored (no-op on re-read)

### Requirement: Refresh Tokens Table

The system MUST create a `refresh_tokens` table for rotation-based token refresh.

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
```

#### Scenario: Token is stored with hash

- GIVEN a user logs in successfully
- WHEN a refresh token is generated
- THEN the SHA-256 hash of the token is stored in `refresh_tokens`
- AND the plaintext token is returned to the client (never stored)

### Requirement: Call Logs Table

(See full spec at `openspec/specs/call-logs/spec.md` for DDL and scenarios.)

### Requirement: FULLTEXT Index on Messages

The system MUST add a FULLTEXT index on `messages.content`.

```sql
CREATE FULLTEXT INDEX ft_messages_content ON messages(content);
```

#### Scenario: Index is created without table rebuild

- GIVEN `messages` table contains existing data
- WHEN the FULLTEXT index DDL runs
- THEN the index is created without dropping or recreating the table
- AND existing messages are indexed after creation (MariaDB builds fulltext in background)
