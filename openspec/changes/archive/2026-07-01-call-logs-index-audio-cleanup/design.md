# Design: Call Logs Index + Audio Constraints Cleanup

## Technical Approach

Two independent micro-optimizations with zero behavioral delta. (1) Add a covering composite index on `call_logs(caller_id, started_at)` so the `ORDER BY started_at DESC` in `RecordCallEnded` is served by the B-tree directly, eliminating a filesort. (2) Hoist the duplicated `audioConstraints` definition in `createPeerConnection` to a single shared variable to eliminate dead copy-paste.

## Architecture Decisions

### Decision: Composite index over functional index or generated column

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `INDEX idx_caller_started (caller_id, started_at)` | Simple DDL, MariaDB can walk (caller_id, started_at) in descending order for the filesort-free plan | **Chosen** |
| Add `INDEX (started_at DESC)` to existing `idx_caller` | Not possible — InnoDB doesn't modify existing indexes; a new index is required anyway | Rejected |
| Rewrite query as two subqueries + UNION | Over-engineered; OR condition already covered by existing `idx_callee` index | Rejected |

**Rationale**: The composite index directly covers the `WHERE caller_id = ? ORDER BY started_at DESC` access path. MariaDB can traverse the B-tree in descending order on `started_at` within each `caller_id` partition — no filesort. The `idx_callee` index already handles the OR's other branch.

### Decision: Hoist variable vs. extract function

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Define `const audioConstraints` once before the try block | ~3 lines moved, minimal diff | **Chosen** |
| Extract `getAudioConstraints()` helper | Overkill for a single use site; caller also mutates `deviceId` inline | Rejected |

**Rationale**: Both branches already construct identical objects. Hoisting eliminates duplication without adding abstraction overhead. The `selectedAudioDevice` mutation is preserved since the hoisted variable is a `const` (mutation of properties, not reassignment — valid).

## Data Flow

```
Item 1 — Query plan change:

  RecordCallEnded ──→ UPDATE call_logs SET ... WHERE caller_id = ?
                         ORDER BY started_at DESC LIMIT 1
                              │
                              ▼
                      idx_caller_started covers both
                      filter (caller_id) + sort (started_at DESC)
                      → No filesort

Item 2 — Variable location:

  Before:   try { const ac = {...}; ... }
            catch { const ac = {...}; ... }   ← duplicate
  After:    const ac = {...};
            try { ... ac ... }
            catch { ... ac ... }              ← single source
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `infra/mariadb/init.sql` | Modify | Add `INDEX idx_caller_started (caller_id, started_at)` after `idx_callee` |
| `client/src/contexts/CallContext.jsx` | Modify | Hoist `audioConstraints` definition before try, remove both inline duplicates |

## Interfaces / Contracts

No new interfaces. The `audioConstraints` object shape is unchanged:

```typescript
interface AudioConstraints {
  echoCancellation: true;
  noiseSuppression: true;
  autoGainControl: true;
  deviceId?: { exact: string };
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| DB schema | Index exists, DDL idempotent | Re-run `init.sql` on existing `call_logs` — no error |
| DB query | No filesort on `RecordCallEnded` query | `EXPLAIN` the UPDATE — verify `Using index`, no `Using filesort` |
| Client | Video fallback still acquires audio-only stream | Manual: deny camera, start video call — verify mic-only stream |
| Client | Audio-only calls unchanged | Manual: start audio call — no regression |

## Migration / Rollout

**DB**: `ALTER TABLE call_logs ADD INDEX IF NOT EXISTS idx_caller_started (caller_id, started_at)` is idempotent. Apply via migration or next init.sql run. Zero downtime — existing queries continue using `idx_caller` until migration completes.

**Client**: Hot-reload safe — React component re-renders pick up the new const reference immediately.

**No data migration required.**

## Open Questions

None — both changes are well-understood, trivial, and carry zero risk.
