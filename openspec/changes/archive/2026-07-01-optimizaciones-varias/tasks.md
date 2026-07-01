# Tasks: Optimizaciones Varias

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~45 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Server — Client.Rooms tracking

- [x] 1.1 Add `Rooms map[string]bool` to `Client` struct in `server/internal/websocket/hub.go`
- [x] 1.2 Update `JoinRoom` to set `client.Rooms[roomID] = true` after adding member
- [x] 1.3 Update `LeaveRoom` to `delete(client.Rooms, roomID)` after removing member
- [x] 1.4 Rewrite `Unregister` to iterate `client.Rooms` instead of scanning all `h.rooms`
- [x] 1.5 Initialize `Rooms: make(map[string]bool)` in `HandleWebSocket` in `handler.go`

## Phase 2: Client — WS exponential backoff

- [x] 2.1 Replace `_reconnect` linear delay with `Math.min(2^(n-1)*1000, 30000) + jitter` in `client/src/lib/websocket.js`

## Phase 3: Client — Remove debug console.log

- [x] 3.1 Remove 5 debug-only `console.log` calls from `client/src/lib/websocket.js`
- [x] 3.2 Remove 21 debug-only `console.log` calls from `client/src/contexts/CallContext.jsx`
- [x] 3.3 Search and remove any remaining debug `console.log` calls in `client/src/`

## Phase 4: Build verification

- [x] 4.1 `go build ./...` in `server/` — no compile errors
- [x] 4.2 `npm run build` in `client/` — no compile errors
