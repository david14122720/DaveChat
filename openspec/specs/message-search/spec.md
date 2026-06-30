# Message Search Specification

## Purpose

FULLTEXT search across messages within a conversation using MariaDB BOOLEAN MODE.

## Requirements

### Requirement: Search Messages

The system MUST expose `GET /api/conversations/{id}/messages/search?q={term}` returning matching messages with relevance.

#### Scenario: Finds matching messages

- GIVEN conversation C has messages containing "urgent", "update", and "meeting"
- WHEN Alice calls `GET /api/conversations/C/messages/search?q=urgent`
- THEN response is HTTP 200 with array of messages containing "urgent"
- AND each result includes `{id, content, sender_id, created_at, relevance}` (FULLTEXT score)

#### Scenario: Empty results

- GIVEN conversation C has no messages matching "zzznotfound"
- WHEN Alice calls `GET /api/conversations/C/messages/search?q=zzznotfound`
- THEN response is HTTP 200 with empty array

#### Scenario: Non-participant gets 403

- GIVEN Alice is NOT a participant in conversation C
- WHEN Alice calls search on C
- THEN response is HTTP 403

### Requirement: FULLTEXT Index

The system MUST create a FULLTEXT index on `messages.content`.

```sql
CREATE FULLTEXT INDEX ft_messages_content ON messages(content);
```

#### Scenario: Index exists after migration

- GIVEN `messages` table exists with `content TEXT`
- WHEN the FULLTEXT index migration runs
- THEN `SHOW INDEX FROM messages` includes `ft_messages_content` with `Index_type = FULLTEXT`

### Requirement: Boolean Mode Query

The system MUST use `MATCH(content) AGAINST(? IN BOOLEAN MODE)` for search.

#### Scenario: Special characters are quoted

- GIVEN user searches for `fix +auth -bug`
- WHEN the handler constructs the query
- THEN each term is wrapped in double quotes to prevent FTS syntax errors
- AND `+` and `-` operators are preserved for boolean mode

#### Scenario: Min token size respected

- GIVEN MariaDB `innodb_ft_min_token_size` defaults to 3
- WHEN user searches for a 2-char term like `go`
- THEN the query returns empty results (MariaDB does not index tokens under min size)
