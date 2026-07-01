# Design: Optimizaciones Varias

## Technical Approach

Three independent single-file changes that improve internal code quality without altering user-facing behavior.

1. **WS backoff** — Replace linear reconnect delay with capped exponential backoff + jitter
2. **Log cleanup** — Remove 26 debug-only `console.log` calls; keep `console.error`/`console.warn`
3. **Room tracking** — Add `Rooms map[string]bool` to `Client` to avoid O(n) room scan on unregister

> **Note**: Item 3 diverges from the proposal/spec (which prescribed a code comment only). The change is small, safe, and justified — rooms are barely used today, but `client.Rooms` costs ~40 bytes per client and eliminates wasteful iteration.

## Architecture Decisions

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Linear backoff (1s,2s,3s…5s) | Simple but predictable thundering-herd on reconnect | **Replace** with exponential + jitter |
| Pure exponential (no cap) | Delay grows to >60s for attempt 6 | **Cap at 30s** — balances aggression with server protection |
| Remove ALL console.log | Risk of deleting meaningful `console.error`/`console.warn` | **Target only console.log** — verified: 26 calls, all debug-only |
| Client.Rooms map vs. comment-only | +1 field per Client, +3 lines of init/modify | **Implement tracking** — O(m) unregister vs O(n), trivial maintenance cost |
| Remove `delete(h.rooms, client.UserID)` attempt | Would be a no-op/nonsense since key is roomID | **Omit** — the proposed outer delete is incorrect for the data structure |

## Data Flow

```
WS backoff:  onclose → _reconnect() → Math.min(2^(n-1)*1000, 30000) + jitter → connect()
Logging:    console.log calls removed → errors still route through console.error/warn
Unregister: iterate client.Rooms → delete from h.rooms[roomID] members → cleanup
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `client/src/lib/websocket.js` | Modify | Replace line 55 with exponential backoff + jitter; remove 5 debug console.log calls (lines 25, 33, 41, 54, 74) |
| `client/src/contexts/CallContext.jsx` | Modify | Remove 21 debug-only console.log calls (lines 216–368) |
| `server/internal/websocket/hub.go` | Modify | Add `Rooms map[string]bool` to Client; update JoinRoom/LeaveRoom/Unregister |
| `server/internal/websocket/handler.go` | Modify | Initialize `Rooms: make(map[string]bool)` in HandleWebSocket |

## Interfaces / Contracts

```go
// Client struct — new field
type Client struct {
    UserID string
    Send   chan []byte
    Hub    *Hub
    Conn   interface{}
    Rooms  map[string]bool      // tracks rooms this client has joined
}

// JoinRoom — after adding to h.rooms:
client.Rooms[roomID] = true

// LeaveRoom — after deleting from h.rooms:
delete(client.Rooms, roomID)

// Unregister — replace room iteration (lines 71-78):
for roomID := range client.Rooms {
    if members, ok := h.rooms[roomID]; ok {
        delete(members, client.UserID)
        if len(members) == 0 {
            delete(h.rooms, roomID)
        }
    }
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Build | Client + server compile | `cd client && npm run build && cd ../server && go build ./...` |
| Manual | WS backoff timing | Browser devtools — disconnect network, verify reconnect timings are exponential + jitter |
| Manual | Console noise | Browser devtools console — verify zero console.log output during normal operation |
| Manual | Hub unregister | Server log — verify no nil pointer or map write panics after client disconnect from a room |

## Migration / Rollout

No migration required. All changes are in-memory (no data, no DB). Room tracking only affects new connections — existing clients without `Rooms` initialized will simply have nil map reads (which return zero value, safe).

## Open Questions

- None. Changes are straightforward, independently revertible, and each commit is self-contained.
