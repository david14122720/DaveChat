# Delta for Internal — Zero-Delta Spec

## Zero-Delta Declaration

This change introduces **zero behavioral changes** to any spec-level requirement. No requirements are added, modified, removed, or renamed. All three items are internal implementation improvements with no user-facing impact.

| Change | Location | Implementation Detail Only |
|--------|----------|---------------------------|
| WS backoff + jitter | `client/src/lib/websocket.js` | Replace linear delay with exponential backoff + jitter capped at 30s |
| Debug log cleanup | `client/src/` (multiple files) | Remove ~26 `console.log` calls; keep `console.error`/`console.warn` in error paths |
| Hub room tracking comment | `server/internal/websocket/hub.go` | Add code comment documenting O(n) room iteration tradeoff |

## Rationale

No existing requirement specifies reconnection delay algorithms, debug output behavior, or code comment style. These are entirely implementation-level concerns:

- **WS backoff**: The signaling spec mandates connection and event relay but does not constrain client retry behavior. The change is a drop-in replacement of the retry timer expression.
- **Log cleanup**: No spec requires or prohibits `console.log` output in development. The removed calls are debug-only artifacts.
- **Hub comment**: No spec mandates code documentation style. The comment is advisory for future maintainers.

## Upstream Diff Impact

None. The signaling spec at `openspec/specs/signaling/spec.md` remains unmodified. No archive merge will touch any domain spec file.
