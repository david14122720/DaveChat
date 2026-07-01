# Proposal: Call Logs Index + Audio Constraints Cleanup

## Intent

Two internal micro-optimizations. (1) `RecordCallEnded` in the WebSocket hub does `ORDER BY started_at DESC` on `call_logs` but only has single-column indexes — MariaDB resorts rows instead of using the index. (2) `audioConstraints` is duplicated verbatim in `CallContext.jsx` inside a catch block, an unnecessary copy-paste.

## Scope

### In Scope
- Add composite index `idx_caller_started` on `call_logs(caller_id, started_at)` to eliminate filesort
- Hoist duplicated `audioConstraints` to a single definition in `createPeerConnection`

### Out of Scope
- Any other call_logs indexes (callee-side, composite status queries)
- Refactoring `createPeerConnection` beyond the duplication fix

## Capabilities

### New Capabilities
None — no new public behavior introduced.

### Modified Capabilities
- `call-logs`: The DDL in spec adds `INDEX idx_caller_started ON call_logs(caller_id, started_at)`. The existing `idx_caller` index remains for queries filtering only by caller_id.

## Approach

**DB**: Add one `INDEX` line in `infra/mariadb/init.sql` after the existing `idx_callee` line. The composite index covers the `WHERE caller_id = ? ORDER BY started_at DESC` pattern directly — MariaDB can walk the B-tree in order without a filesort.

**Client**: Define `const audioConstraints = {...}` once after `let stream;` in `createPeerConnection` (line 163), then reference the same variable in both the main `try` block and the video-error `catch` fallback. ~5 lines.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `infra/mariadb/init.sql` | Modified | Add `idx_caller_started` composite index |
| `client/src/contexts/CallContext.jsx` | Modified | Deduplicate `audioConstraints` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Composite index unused if query plan chooses different path | Low | Verify with `EXPLAIN` — MariaDB prefers covering indexes for ORDER BY |
| Shadow variable bug if hoisting breaks scope | Low | Single definition before both usages; both paths already have identical constraints |

## Rollback Plan

- **DB**: Drop the added index line — the existing `idx_caller` still works
- **Client**: Revert the hoisting — both paths already work independently

## Dependencies

None.

## Success Criteria

- [ ] `infra/mariadb/init.sql` applies idempotently with new index
- [ ] `EXPLAIN` on the `RecordCallEnded` query shows `Using index` without `Using filesort`
- [ ] Video call fallback (no camera) still acquires audio-only stream
