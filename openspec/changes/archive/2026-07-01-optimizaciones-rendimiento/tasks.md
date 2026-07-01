# Tasks: Performance Optimizations

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~25-35 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | single-pr |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: single-pr
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Server throttle + Client memoization | PR 1 | Single PR — both are independent, trivially reviewable |

## Phase 1: Server — Throttle TouchPresence DB writes

- [x] 1.1 Add `lastPresenceTouch map[string]time.Time` field to `Hub` struct in `server/internal/websocket/hub.go`
- [x] 1.2 Initialize map in `NewHub()`: `lastPresenceTouch: make(map[string]time.Time)`
- [x] 1.3 Guard `TouchPresence()`: return early if `lastPresenceTouch[userID] < 30s ago`
- [x] 1.4 Update `lastPresenceTouch[userID] = time.Now()` before calling `setPresence`
- [x] 1.5 Cleanup entry in `Unregister()`: `delete(h.lastPresenceTouch, userID)`

## Phase 2: Client — useMemo in CallContext value

- [x] 2.1 Import `useMemo` from `react` in `client/src/contexts/CallContext.jsx`
- [x] 2.2 Wrap the `value` object in `useMemo` with deps: `[state, startCall, acceptCall, rejectCall, endCall, resetCall, audioDevices, selectedAudioDevice, setSelectedAudioDevice]`

## Phase 3: Verification — Build check

- [x] 3.1 Run `cd server && go build ./...` — confirm server compiles
- [x] 3.2 Run `cd client && npm run build` — confirm client builds
