# Delta for Call Logs

## Overview

This change is **internal-only with zero behavioral delta**. Every requirement, scenario, and GIVEN/WHEN/THEN in the existing spec remains identical after implementation. The DDL adds a covering composite index — queries produce the same results faster. The client change hoists a duplicate `const audioConstraints` into a single definition — both paths already produce identical constraints.

## ADDED Requirements

None — no new observable behavior is introduced.

## MODIFIED Requirements

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
CREATE INDEX idx_call_logs_caller_started ON call_logs(caller_id, started_at);
```
(Previously: DDL had only `idx_caller` and `idx_callee` single-column indexes)

#### Scenario: Schema applies idempotently

- GIVEN `call_logs` table already exists
- WHEN the init SQL runs again
- THEN no error occurs (IF NOT EXISTS)

#### Scenario: Composite index covers ORDER BY query

- GIVEN the new schema has been applied
- WHEN a query runs `SELECT ... FROM call_logs WHERE caller_id = ? ORDER BY started_at DESC`
- THEN `EXPLAIN` shows `Using index` without `Using filesort`

## REMOVED Requirements

None.

## RENAMED Requirements

None.

## Non-Spec Changes

The following change has **no spec-level impact** — it is purely internal code deduplication:

- **File**: `client/src/contexts/CallContext.jsx`
- **What**: `const audioConstraints` defined once after `let stream;` in `createPeerConnection`, replacing an identical duplicate inside the catch block
- **Why**: Both the main `try` block and the video-error `catch` fallback used the same constraints — hoisting eliminates dead duplication
- **Behavioral delta**: Zero — both paths produce identical `MediaStreamConstraints` before and after
