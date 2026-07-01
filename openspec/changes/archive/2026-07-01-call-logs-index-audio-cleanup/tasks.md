# Tasks: Call Logs Index + Audio Constraints Cleanup

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~10–15 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Schema & Code Changes

- [x] 1.1 `infra/mariadb/init.sql` — Add `INDEX idx_caller_started (caller_id, started_at)` after `INDEX idx_callee (callee_id)` (line 81)
- [x] 1.2 `client/src/contexts/CallContext.jsx` — Hoist `const audioConstraints = {...}` to right after `let stream;`, remove both duplicate definitions inside main try and video catch blocks

## Phase 2: Build Verification

- [x] 2.1 Verify Go build: `cd server && go build ./...`
- [x] 2.2 Verify client build: `cd client && npm run build`
