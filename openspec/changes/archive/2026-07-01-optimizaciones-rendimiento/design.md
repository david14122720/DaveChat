# Design: Performance Optimizations

## Technical Approach

Two isolated internal optimizations — zero behavioral changes, pure resource reduction.
- **Server**: Throttle `TouchPresence` DB writes to ≤1 per 30s per user
- **Client**: Memoize `CallContext.Provider` value to eliminate cascading re-renders

## Architecture Decisions

### Decision: In-memory throttle map vs. Redis/timestamp column

| Option | Tradeoff | Decision |
|--------|----------|----------|
| In-memory `map[string]time.Time` | Simple, zero infra; lost on restart (acceptable — in-flight touches are non-critical) | **Chosen** |
| DB `last_seen` column re-read | Redundant DB call defeats optimization purpose | Rejected |
| Redis TTL | Overkill for transient presence throttle | Rejected |

**Rationale**: The throttle is a transient rate limit, not authoritative state. Ephemeral in-memory tracking is correct — on restart, the worst case is one extra write per user.

### Decision: Cleanup strategy for throttle map

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Cleanup in Unregister only | Simple; users reconnect daily, map never leaks indefinitely | **Chosen** |
| Periodic goroutine cleanup | More robust for long-lived connections, but adds complexity | Rejected |

**Rationale**: Users disconnect frequently. Unregister already runs per-client and has the lock. Adding a goroutine means either a new `time.Ticker` or a cancellable goroutine in Hub lifecycle, neither of which exists today. The Unregister-only approach is sufficient — even if a client stays connected for days, the map has one entry per connected user, which is bounded by active clients.

### Decision: useMemo dependency list — explicit state keys vs. full state

**Choice**: Explicitly list `state` (object identity) plus stable callbacks
**Rationale**: Spreading `state` creates a new object each render. `useMemo` with `[state, ...]` breaks the identity chain. All callbacks (`startCall`, `acceptCall`, etc.) are `useCallback`-wrapped and stable. `audioDevices`/`selectedAudioDevice`/`setSelectedAudioDevice` are `useState` values, stable between changes.

## Data Flow

**TouchPresence (server)**:
```
WebSocket message → TouchPresence(userID)
     │
     ├─ lock + check online + check lastPresenceTouch[userID]
     │   └─ if < 30s since last → unlock + return (skip DB)
     │
     └─ update lastPresenceTouch[userID] + unlock
         └─ setPresence(userID, "online")
              └─ DB: UPDATE profiles SET status=?, last_seen=NOW(3)
```

**CallContext (client)**:
```
dispatch({ type }) → reducer → new state object
     │
     └─ useMemo([state, ...callbacks, ...audioState])
          └─ value identity unchanged if deps haven't changed
               └─ children consuming useCall() skip re-render
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `server/internal/websocket/hub.go` | Modify | Add `lastPresenceTouch map[string]time.Time` to Hub; initialize in NewHub; guard in TouchPresence; cleanup in Unregister |
| `client/src/contexts/CallContext.jsx` | Modify | Import `useMemo`; wrap `value` object in `useMemo` with explicit dependency list |

## Interfaces / Contracts

No new interfaces. No behavioral contract changes.

**Hub struct** — one new field:
```go
type Hub struct {
    // ... existing fields
    lastPresenceTouch map[string]time.Time
}
```

**TouchPresence** — logical flow (pseudocode):
```
func (h *Hub) TouchPresence(userID string) {
    h.mu.Lock()
    if _, online := h.clients[userID]; !online {
        h.mu.Unlock()
        return
    }
    if last, ok := h.lastPresenceTouch[userID]; ok && time.Since(last) < 30*time.Second {
        h.mu.Unlock()
        return
    }
    h.lastPresenceTouch[userID] = time.Now()
    h.mu.Unlock()
    h.setPresence(userID, "online")
}
```

**CallContext value** — wrap in useMemo:
```js
const value = useMemo(() => ({
  ...state,
  startCall, acceptCall, rejectCall, endCall, resetCall,
  audioDevices, selectedAudioDevice, setSelectedAudioDevice,
  isIdle: state.callState === CallState.IDLE,
  isCalling: state.callState === CallState.CALLING,
  isRinging: state.callState === CallState.RINGING,
  isConnected: state.callState === CallState.CONNECTED,
}), [state, startCall, acceptCall, rejectCall, endCall, resetCall,
    audioDevices, selectedAudioDevice, setSelectedAudioDevice]);
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | TouchPresence throttle | Manual verify: start 10 rapid messages, confirm 1 DB write. No test infra exists |
| Unit | CallContext memoization | Manual verify: React DevTools profiler shows Provider value stabilizes |
| Build | Both files compile | `cd client && npm run build` / `go build ./...` |

## Migration / Rollout

No migration required. Both changes are hot-swappable:
- **Server**: Deploy updated binary; in-flight connections continue with slight miss on first post-deploy TouchPresence (map is empty after restart — one extra write per user)
- **Client**: Rebuild and serve; browser gets new JS on next load

## Open Questions

None.
