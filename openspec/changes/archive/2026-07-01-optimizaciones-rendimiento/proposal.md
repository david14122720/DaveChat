# Proposal: Performance Optimizations

## Intent

Reduce unnecessary DB writes and React re-renders. TouchPresence hits MariaDB on every WebSocket message, and CallContext value recreates on every dispatch causing cascading re-renders. Both are invisible to users today but waste resources.

## Scope

### In Scope
- Throttle `TouchPresence` DB writes to max 1 per 30s per user
- Wrap `CallContext.Provider` value in `useMemo` with proper deps

### Out of Scope
- Ringtone AudioContext (already correct)
- `newUUID` manual impl (already correct)
- Unregister room iteration (low impact)
- console.log cleanup (cosmetic)
- WS backoff (low impact)
- DB indexes (separate effort)

## Capabilities

### New Capabilities
None — pure internal optimization, no new capabilities.

### Modified Capabilities
None — neither change alters spec-level requirements or public-facing behavior.

## Approach

**Server**: Add `lastPresenceTouch map[string]time.Time` to Hub struct (alongside existing maps). Before `setPresence` DB write, skip if last touch was <30s ago. ~15-20 lines in hub.go.

**Client**: Wrap CallContext.Provider `value` in `useMemo` with explicit dependency list matching used state. ~5 lines in CallContext.jsx.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| server/internal/websocket/hub.go | Modified | Add throttle map + guard in TouchPresence |
| client/src/contexts/CallContext.jsx | Modified | Add useMemo around provider value |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Missed presence update during throttle window | Low | 30s is acceptable for "online" accuracy — last_seen has second granularity |
| Stale closure in useMemo deps | Low | List all used state keys explicitly; will catch in code review |

## Rollback Plan

- **Server**: Revert hub.go changes, redeploy
- **Client**: Remove useMemo wrapper, commit revert

## Dependencies

None.

## Success Criteria

- [ ] MariaDB `UPDATE profiles SET status = 'online'...` rate drops to ≤1 per 30s per active user
- [ ] Components consuming `useCall()` no longer re-render when unrelated CallContext dispatches fire
