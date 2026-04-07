---
phase: 04-command-ux-and-migration-help
plan: 01
subsystem: cli
tags: [doctor, migration-prompt, layout-resolution, authority, go]
requires:
  - phase: 03-authority-and-safety
    provides: typed authority checks, sibling warnings, and fail-loud resolver status
provides:
  - Resolver-backed doctor inspection for global, repo, and all scopes
  - Shared doctor result model that drives concise, explain, and JSON output
  - Advisory migration briefs with deterministic path and inventory facts
affects: [04-command-ux-and-migration-help, 05-rollout-verification]
tech-stack:
  added: []
  patterns: [shared doctor result for all output tiers, advisory markdown migration briefs, CLI-boundary migration guidance]
key-files:
  created:
    - internal/app/doctor.go
    - internal/app/doctor_test.go
  modified:
    - internal/app/app.go
    - internal/cli/app.go
    - internal/cli/app_integration_test.go
key-decisions:
  - "The app layer returns one structured doctor result that the CLI reuses for concise, explain, and JSON rendering."
  - "Migration prompts stay advisory-only and include deterministic path metadata plus inventories without reading file bodies."
  - "Ambiguous authority failures gain migration-prompt guidance only at the CLI boundary so config-layer authority typing stays unchanged."
patterns-established:
  - "Doctor inspects resolver state directly instead of calling ordinary-operation config loading, so legacy-only and invalid-manifest scopes stay visible."
  - "Operator-facing migration help is generated from structured inspection data, not from a separate ad hoc resolver path."
requirements-completed: [GLBL-03, AUTH-03, MIGR-02, MIGR-03]
duration: 12m34s
completed: 2026-04-07
---

# Phase 4 Plan 1: Command UX and Migration Help Summary

**Resolver-backed `changes doctor` inspection, JSON/explain rendering, and advisory migration briefs for authoritative-layout cleanup**

## Performance

- **Duration:** 12m34s
- **Started:** 2026-04-07T02:40:09Z
- **Completed:** 2026-04-07T02:52:43Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Added an app-layer doctor service that inspects global and repo layout state directly from resolver and authority contracts.
- Added a single CLI `doctor` command family with concise, `--explain`, `--json`, and `--migration-prompt` modes on the approved grammar.
- Extended ambiguity-facing CLI failures to point operators toward both inspection and migration-brief workflows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the doctor inspection and migration-prompt application service** - `c3d1556` (test), `4862aff` (feat)
2. **Task 2: Wire `changes doctor` into the CLI with flag-driven help, output, and ambiguity hints** - `dc74f52` (test), `6987f0b` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified

- `internal/app/app.go` - Added shared doctor request/result types and JSON-facing structures.
- `internal/app/doctor.go` - Implemented resolver-backed inspection, repair hints, deterministic inventories, and advisory migration brief generation.
- `internal/app/doctor_test.go` - Locked the doctor service contract and migration brief content with focused app tests.
- `internal/cli/app.go` - Routed the `doctor` command, rendered concise/explain/JSON output, handled migration prompt file/stdout behavior, and extended ambiguity errors with migration guidance.
- `internal/cli/app_integration_test.go` - Added CLI regression coverage for doctor defaults, help text, output modes, prompt file behavior, and ambiguity hints.

## Decisions Made

- Reused one structured doctor result across all output tiers so the CLI does not need separate resolver-specific rendering paths for concise, explain, and JSON modes.
- Kept migration help advisory and deterministic: prompts enumerate paths, schema metadata, and inventories, but never read or embed artifact contents.
- Preserved the typed `AuthorityError` contract and added the migration-brief suggestion only where the CLI formats failures.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Synced stale planning body text after automated state updates**
- **Found during:** Final metadata update
- **Issue:** `gsd-tools` updated structured plan progress and requirement state, but the human-readable sections in `STATE.md`, `ROADMAP.md`, and `REQUIREMENTS.md` still showed pre-execution values.
- **Fix:** Manually updated the stale progress, roadmap, and last-updated lines so the planning artifacts matched the completed plan.
- **Files modified:** `.planning/STATE.md`, `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`
- **Verification:** Re-read all three files and confirmed Phase 04 Plan 01 shows complete, roadmap progress is `1/2`, and the completed requirements are checked.
- **Committed in:** final docs commit

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope change. The fix kept the execution metadata internally consistent after the plan completed.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 4 now has the doctor inspection and migration workflow in place for operators and for later rollout verification.
- Ready for Plan 04-02 to finish init UX, path-reporting, and documentation work on top of the locked doctor surface.

## Self-Check: PASSED

- Found `.planning/phases/04-command-ux-and-migration-help/04-command-ux-and-migration-help-01-SUMMARY.md`.
- Verified task commits `c3d1556`, `4862aff`, `dc74f52`, and `6987f0b` exist in `git log`.
