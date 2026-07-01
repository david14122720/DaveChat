# Spec: Performance Optimizations

> **Zero behavioral delta** — both changes are internal optimizations with no user-facing effect, no API changes, and no spec-level requirements changes.

## Capability Impact

| Capability | Status | Rationale |
|------------|--------|-----------|
| All existing capabilities | Unchanged | Neither throttled TouchPresence writes nor useMemo wrapping alter observable behavior |

No `openspec/specs/*/spec.md` files are modified by this change. No new capability specs are created.

## ADDED Requirements

None — no new capabilities are introduced.

## MODIFIED Requirements

None — no existing capability's requirements are altered. The `TouchPresence` behavior remains identical from the caller's perspective:

- **Throttle guard**: The DB write is skipped when `<30s` since last touch, but the in-memory presence state is still updated. Downstream consumers see the same results.
- **useMemo wrapper**: The `value` object identity changes less frequently, but the rendered output is identical. Components get the same data, just fewer re-renders.

## REMOVED Requirements

None.

## RENAMED Requirements

None.

## Verification

Both optimizations are verified via existing runtime behavior (no behavioral change) plus proposal-level success criteria:

- [ ] MariaDB `UPDATE profiles SET status = 'online'...` rate drops to ≤1 per 30s per active user
- [ ] Components consuming `useCall()` no longer re-render when unrelated CallContext dispatches fire

See `proposal.md` for technical approach and affected files.
