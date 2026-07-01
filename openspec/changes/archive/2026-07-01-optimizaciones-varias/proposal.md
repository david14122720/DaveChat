# Proposal: Optimizaciones Varias

## Intent

Bundle three low-risk internal optimizations that improve code quality and reliability without changing user-facing behavior. Each is independently verifiable and trivially revertible.

## Scope

### In Scope
- **WS backoff**: Replace linear reconnect delay (1s, 2s, 3s…) with exponential backoff + jitter capped at 30s in `client/src/lib/websocket.js`
- **Log cleanup**: Remove ~26 debug-only `console.log` calls across `client/src/` (CallContext.jsx, websocket.js, others). Keep `console.error`/`console.warn` in error paths.
- **Hub room cleanup note**: Acknowledge the O(n) room iteration pattern in `server/internal/websocket/hub.go` Unregister; leave as-is since rooms are barely used. Add a code comment documenting the tradeoff.

### Out of Scope
- No behavioral changes or new features
- No refactoring beyond the targeted lines
- No Client.rooms tracking map (deferred unless rooms become active)
- No test setup or CI changes

## Capabilities

### New Capabilities
None — no new spec-level capabilities introduced.

### Modified Capabilities
None — no spec-level requirements change for any existing capability. All three items are internal implementation changes with zero behavioral impact.

## Approach

Three independent, single-file changes:

1. **WS backoff** (`client/src/lib/websocket.js`, ~3 lines): Replace `1000 * this.reconnectAttempts` with `Math.min(1000 * Math.pow(2, n-1), 30000) + Math.random() * 1000`
2. **Log cleanup** (`client/src/`): Search all `.js`/`.jsx` files for `console.log`, review each, remove debug-only lines, keep error/warning logging.
3. **Hub note** (`server/internal/websocket/hub.go`): Add a short comment above the room iteration loop documenting the O(n) pattern and why it's acceptable.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `client/src/lib/websocket.js` | Modified | Exponential backoff + jitter for reconnection |
| `client/src/contexts/CallContext.jsx` | Modified | Remove ~12 debug console.log calls |
| `client/src/lib/websocket.js` | Modified | Remove ~5 debug console.log calls |
| `client/src/` (other files) | Modified | Remove ~9 debug console.log calls |
| `server/internal/websocket/hub.go` | Modified | Add comment documenting room iteration tradeoff |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Accidentally removing a meaningful console.warn/error | Low | Target only `console.log` — keep `console.error` and `console.warn` |
| Reconnection timing change causes race | Low | Jitter stays within +1s of exponential schedule; existing timeout logic unchanged |
| Hub comment is noise | Low | Comment documents intentional design choice, prevents future "optimization" churn |

## Rollback Plan

Each change is in a separate commit. Revert individual commits or files as needed:
- `git revert <ws-backoff-commit>`
- `git revert <log-cleanup-commit>`
- `git revert <hub-comment-commit>` (or discard the comment manually)

## Dependencies

None. All changes are self-contained.

## Success Criteria

- [ ] Client builds and runs without new errors (`cd client && npm run build`)
- [ ] Server builds successfully (`cd server && go build ./...`)
- [ ] WebSocket reconnects with exponential backoff (verify in browser devtools network tab on disconnect)
- [ ] No `console.log` output remains in normal client operation (verify in browser devtools console)
- [ ] Hub.go has a comment documenting the room iteration pattern
