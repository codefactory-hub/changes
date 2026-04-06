---
phase: 02-resolution-core
plan: 02
subsystem: config
tags: [go, config, resolver, init, xdg, home]
requires:
  - phase: 02-resolution-core
    provides: Resolver contracts, candidate statuses, and manifest parsing from Plan 01
provides:
  - Resolver-backed compatibility helpers and repo config loading
  - Deterministic repo-init layout selection with explicit precedence
  - Init flow wiring that stamps managed layouts and selected state ignore entries
affects: [03-authority-and-safety, 04-command-ux-and-migration-help, 05-rollout-verification]
tech-stack:
  added: []
  patterns: [resolver-backed compatibility wrappers, deterministic repo-init selection, init-time layout manifest stamping]
key-files:
  created:
    - internal/config/init_defaults.go
    - internal/config/init_defaults_test.go
  modified:
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/app/app.go
    - internal/app/init.go
    - internal/app/app_test.go
key-decisions:
  - "Compatibility helpers read authoritative resolver paths and only fall back to explicit config overrides when callers have diverged from built-in defaults."
  - "Repo initialization chooses a repo-local layout through one config package helper and writes a matching layout manifest so resolver-backed loads remain operational."
  - "Init request inputs carry optional layout and home selections without forcing broader CLI rewiring in this phase."
patterns-established:
  - "Compatibility Pattern: exported config/path helpers stay stable while resolver candidate paths become the source of truth."
  - "Bootstrap Pattern: uninitialized repos select layout once, then write config.toml, layout.toml, and gitignore state entries from that shared result."
requirements-completed: [GLBL-01, GLBL-02, REPO-01, REPO-03]
duration: 8 min
completed: 2026-04-06
---

# Phase 2 Plan 02: Resolution Core Summary

**Resolver-backed config loading, deterministic repo-init layout selection, and init-path manifest stamping for managed xdg and home layouts**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-06T22:46:52Z
- **Completed:** 2026-04-06T22:55:20Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments
- Rebased `internal/config` helper functions and `Load` onto authoritative resolver results while preserving custom path overrides already expressed in repo config.
- Added a deterministic `SelectRepoInitLayout` API that applies flags, global defaults, `CHANGES_HOME`, XDG env signals, and built-in defaults in the approved order.
- Wired `Initialize` to the shared selection helper, selected `.gitignore` state entries, and manifest-backed bootstrap outputs so newly initialized repos stay loadable under the resolver contract.

## Task Commits

Each task was committed atomically:

1. **Task 1: Rebase config helpers and config loading onto the resolver core** - `581100d` (test), `c282eba` (feat)
2. **Task 2: Add deterministic repo-init selection helpers for the uninitialized case** - `c4b8916` (test), `47ed39b` (feat)
3. **Task 3: Wire the current init flow to the shared repo-init selection helper** - `0bd9b42` (test), `975bd2f` (feat)

## Files Created/Modified
- `internal/config/config.go` - Resolver-backed compatibility wrappers and resolver-status-aware config loading.
- `internal/config/config_test.go` - Regression coverage for authoritative repo config lookup, helper path selection, and uninitialized init hints.
- `internal/config/init_defaults.go` - Deterministic repo-init selection API with precedence and repo-home path validation.
- `internal/config/init_defaults_test.go` - TDD coverage for default xdg selection, repo-home defaults, and `CHANGES_HOME` precedence over XDG signals.
- `internal/app/app.go` - Optional init request fields for requested layout and requested home path.
- `internal/app/init.go` - Shared init-layout selection, managed layout manifest writing, and selected `.gitignore` updates.
- `internal/app/app_test.go` - Regression coverage proving the real init flow honors selected home layouts and state ignore entries.

## Decisions Made
- Kept the exported config/path helper signatures unchanged and pushed the layout decision behind resolver-backed helper internals.
- Treated repo-init layout selection as config policy, not CLI policy, so later `init`, `init global`, and `doctor` work can reuse one precedence implementation.
- Wrote `layout.toml` during repo initialization once the resolver-backed loader started rejecting `legacy_only` layouts.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Initialize now writes `layout.toml` alongside `config.toml`**
- **Found during:** Task 3 (Wire the current init flow to the shared repo-init selection helper)
- **Issue:** After Task 1, `config.Load` correctly rejects `legacy_only` layouts. Without manifest stamping during init, a freshly initialized repo would immediately become non-loadable.
- **Fix:** Added init-time layout manifest generation from the selected repo layout and wrote it transactionally next to the selected config directory.
- **Files modified:** `internal/app/init.go`
- **Verification:** `go test ./internal/app -count=1`; `go test ./...`
- **Committed in:** `975bd2f` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The auto-fix kept the plan’s resolver-backed bootstrap path operational without expanding scope beyond the init flow already under change.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Resolver-backed compatibility helpers, repo-init precedence, and managed init outputs are in place for Phase 3 authority and safety work.
- Newly initialized repos now carry both config and structural layout metadata, so later ambiguity and diagnostic work can assume manifest-backed bootstrap behavior.

## Self-Check: PASSED
