---
phase: 03-authority-and-safety
plan: 01
subsystem: config
tags: [go, config, authority, resolver, manifests, safety]
requires:
  - phase: 02-resolution-core
    provides: Resolver contracts, candidate statuses, manifest parsing, and init-time manifest stamping
provides:
  - Authority grouping that distinguishes real ambiguity from canonical-equivalent candidates
  - Structured sibling warnings for legacy-only and invalid-manifest candidates when one authority remains
  - Shared authority-check helpers with typed statusful failures and terse doctor guidance
affects: [03-authority-and-safety, 04-command-ux-and-migration-help, 05-rollout-verification]
tech-stack:
  added: []
  patterns: [authority-grouped resolution, structured authority warnings, typed config-layer authority errors]
key-files:
  created:
    - internal/config/authority.go
    - internal/config/authority_test.go
  modified:
    - internal/config/resolution_types.go
    - internal/config/resolution.go
    - internal/config/manifest.go
    - internal/config/resolution_test.go
    - internal/config/manifest_test.go
key-decisions:
  - "Operational authority is determined by canonical authority groups instead of raw resolved-candidate count."
  - "Resolved scopes keep structured sibling warnings so later CLI work can render them without re-deriving policy."
  - "Authority failures stay typed and terse, with doctor as the next-step command for unresolved scopes."
patterns-established:
  - "Authority Pattern: resolve scope first, then call CheckScopeAuthority before ordinary read or write operations."
  - "Manifest Pattern: unsupported schema versions are operationally invalid rather than repaired during normal resolution."
requirements-completed: [AUTH-01, AUTH-02]
duration: 6 min
completed: 2026-04-07
---

# Phase 3 Plan 01: Authority and Safety Summary

**Authority-aware repo resolution with canonical equivalence collapse, strict manifest schema gating, and shared typed authority checks**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-07T01:22:48Z
- **Completed:** 2026-04-07T01:29:10Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Reworked resolver status calculation around canonical authority groups so distinct operational layouts still fail loudly while equivalent candidates collapse to one authority.
- Preserved structured warning data on resolved scopes when legacy-only or invalid-manifest siblings coexist with one authoritative candidate.
- Added reusable authority-check helpers that return typed failures with concise `changes doctor --scope ...` guidance for later command wiring.

## Task Commits

Each task was committed atomically:

1. **Task 1: Tighten resolver authority semantics, sibling warnings, and schema-version validity** - `d32ce0d` (test), `1e61cf2` (feat)
2. **Task 2: Add shared authority-check helpers with terse doctor-guided failure messages** - `4576d0c` (test), `5705bc0` (feat)

_Note: This plan used TDD, so each task produced a RED test commit and a GREEN implementation commit._

## Files Created/Modified
- `internal/config/resolution_types.go` - Extended scope resolution results to carry structured authority warnings.
- `internal/config/resolution.go` - Added authority-grouped resolution, authoritative-candidate selection, and sibling warning collection.
- `internal/config/manifest.go` - Enforced exact supported `schema_version` validity for operational manifests.
- `internal/config/resolution_test.go` - Added resolver coverage for ambiguity, mixed sibling warnings, and canonical-equivalent candidate collapse.
- `internal/config/manifest_test.go` - Added strict unsupported-schema rejection coverage without manifest rewrites.
- `internal/config/authority.go` - Added typed authority warnings, check results, error classification, and terse doctor-guided messages.
- `internal/config/authority_test.go` - Added coverage for ambiguous, legacy-only, invalid-manifest, and warning-preserving authority checks.

## Decisions Made
- Used canonical root grouping to distinguish true competing authorities from candidates that resolve to the same physical target through symlinks.
- Kept warning rendering out of `internal/config`; lower layers now return structured warning values only.
- Preserved competing candidates on `AuthorityError` so later doctor and CLI layers can render richer diagnostics without string parsing.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/config` now exposes the authority policy surface needed to wire config, init, and CLI entry points to one shared no-dual-write gate.
- Later doctor and migration work can reuse `AuthorityError`, `AuthorityCheck`, and scope warnings without duplicating resolver policy.

## Self-Check: PASSED
