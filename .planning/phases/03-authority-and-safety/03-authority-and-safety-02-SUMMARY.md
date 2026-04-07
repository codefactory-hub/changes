---
phase: 03-authority-and-safety
plan: 02
subsystem: cli
tags: [go, cli, config, authority, safety]
requires:
  - phase: 03-authority-and-safety
    provides: Shared authority checks, typed authority failures, and structured sibling warnings
provides:
  - Warning-aware config loading for repo callers
  - Explicit repo write-authority gates at init, create, and release entry points
  - CLI stderr rendering for repo and global authority warnings
affects: [03-authority-and-safety, 04-command-ux-and-migration-help, 05-rollout-verification]
tech-stack:
  added: []
  patterns: [warning-aware config loads, entry-point write gates, cli stderr authority warnings]
key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/app/app.go
    - internal/app/init.go
    - internal/app/app_test.go
    - internal/cli/create.go
    - internal/cli/app.go
    - internal/cli/app_integration_test.go
key-decisions:
  - "Load remains a compatibility wrapper over LoadWithAuthority so unchanged callers still compile while new flows preserve warnings."
  - "Initialize only treats genuinely uninitialized global authority as 'no defaults'; ambiguous, legacy-only, and invalid global states now surface typed authority errors."
  - "Authority warning presentation stays in internal/cli, with repo paths rendered relative to the repo root and global paths left absolute."
patterns-established:
  - "Config Pattern: ordinary repo reads use LoadWithAuthority when warning propagation matters, while Load delegates to the same authority-checked path."
  - "Write Gate Pattern: each mutating entry point calls RequireRepoWriteAuthority explicitly before managed disk writes."
requirements-completed: [AUTH-01, AUTH-02, MIGR-04]
duration: 13 min
completed: 2026-04-07
---

# Phase 3 Plan 02: Authority and Safety Summary

**Authority-aware config/app/cli entry points with stderr warning propagation and explicit no-dual-write gates**

## Performance

- **Duration:** 13 min
- **Started:** 2026-04-07T01:35:24Z
- **Completed:** 2026-04-07T01:48:20Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments
- Added warning-aware repo config loading plus an explicit shared repo write-authority helper in `internal/config`.
- Propagated structured authority warnings through `init`, `status`, `release`, and `render`, and enforced repo/global authority checks before init and release writes.
- Rendered repo/global authority warnings on CLI stderr and gated fragment creation with an explicit repo write-authority check.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add warning-aware config loading and explicit repo write-authority helpers** - `c507d5c` (test), `ef0f203` (feat)
2. **Task 2: Propagate authority warnings through app services and recheck write gates before disk mutations** - `377dceb` (test), `2c34729` (feat)
3. **Task 3: Surface authority warnings on stderr and apply explicit create-path write gating in the CLI** - `8d7c1f6` (test), `9e4abee` (feat)

_Note: This plan used TDD, so each task produced a RED test commit and a GREEN implementation commit._

## Files Created/Modified
- `internal/config/config.go` - Added `LoadWithAuthority`, `RequireRepoWriteAuthority`, and the shared authority-checked repo config load path.
- `internal/config/config_test.go` - Added coverage for warning-preserving loads and write-gate failures across legacy, invalid, and ambiguous repo layouts.
- `internal/app/app.go` - Propagated authority warnings through app result structs and rechecked repo write authority before release record writes.
- `internal/app/init.go` - Switched repo/global init authority decisions to shared `CheckScopeAuthority` handling and merged global sibling warnings into `InitializeResult`.
- `internal/app/app_test.go` - Added service-layer coverage for warning propagation and fail-loud init/release authority behavior.
- `internal/cli/create.go` - Changed `create` to use warning-aware config loading and an explicit repo write gate before fragment creation.
- `internal/cli/app.go` - Added CLI-only stderr warning rendering for repo/global authority warnings and wired it through init/status/release/render flows.
- `internal/cli/app_integration_test.go` - Added end-to-end CLI coverage for stderr warnings and terse authority failures.

## Decisions Made
- Kept `Load` as a compatibility wrapper instead of widening every existing config call site at once; warning-aware flows now opt into `LoadWithAuthority`.
- Returned typed `AuthorityError` values unchanged from app/init authority checks so CLI error output stays terse and doctor-guided.
- Printed structured warnings only in the CLI layer, preserving lower layers as data-returning policy code rather than presentation code.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Repo and global authority warnings now arrive at the CLI as structured data, which gives Phase 4 doctor and migration UX work a stable presentation boundary.
- All managed write entry points in scope now recheck repo authority explicitly, so future command UX work can build on a consistent no-dual-write gate.

## Self-Check: PASSED
