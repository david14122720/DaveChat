# Archive Report: Call Logs Index + Audio Constraints Cleanup

**Archived on**: 2026-07-01
**Archive path**: `openspec/changes/archive/2026-07-01-call-logs-index-audio-cleanup/`

## Intent

Two internal micro-optimizations:
1. Add composite INDEX `idx_caller_started (caller_id, started_at)` to `call_logs` table to eliminate filesort in `RecordCallEnded`
2. Hoist duplicated `audioConstraints` definition in `CallContext.jsx` to a single shared variable

## Tasks Gate

All 4 tasks marked `[x]` — no stale unchecked items. Gate passed.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| call-logs | Modified | 1 requirement modified (Call Log Table Schema — added composite index + new scenario) |

## Source of Truth Updated

- `openspec/specs/call-logs/spec.md` — DDL now includes `CREATE INDEX idx_call_logs_caller_started ON call_logs(caller_id, started_at)` and new "Composite index covers ORDER BY query" scenario

## Archive Contents

- proposal.md ✅
- specs/call-logs/spec.md ✅ (delta spec preserved)
- design.md ✅
- tasks.md ✅ (4/4 tasks complete)
- archive-report.md ✅ (this file)

## Intentional Partial Archive

**Missing artifact**: `verify-report.md` — no verify report file was generated during the SDD cycle. The user explicitly confirmed "All tasks verified, builds pass. Archive the change." Archive proceeding with explicit user approval for this intentional partial archive.

## Delivered Changes

| File | Change |
|------|--------|
| `infra/mariadb/init.sql` | Added `INDEX idx_caller_started (caller_id, started_at)` after `idx_callee` |
| `client/src/contexts/CallContext.jsx` | Hoisted `const audioConstraints` to single definition before try block |
