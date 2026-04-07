---
phase: 04-command-ux-and-migration-help
plan: 02
subsystem: cli-docs
tags: [init, global-init, docs, layout-resolution, go]
requires:
  - phase: 04-command-ux-and-migration-help
    provides: doctor inspection, migration-prompt CLI workflow, and approved doctor grammar
provides:
  - Minimal repo/global init layout-selection UX on the approved command shapes
  - Explicit init success reporting for selected layout plus resolved config/data/state paths
  - Default-first README documentation for init, doctor, precedence, and repo-local state ignore behavior
affects: [04-command-ux-and-migration-help, 05-rollout-verification]
tech-stack:
  added: []
  patterns: [shared init result metadata for CLI rendering, authoritative state-ignore reporting, default-first operator documentation]
key-files:
  created: []
  modified:
    - internal/config/init_defaults.go
    - internal/config/init_defaults_test.go
    - internal/app/app.go
    - internal/app/init.go
    - internal/app/init_test.go
    - internal/cli/app.go
    - internal/cli/app_integration_test.go
    - README.md
key-decisions:
  - "Repo and global init share the same minimal `--layout` and `--home` grammar, with `--home` rejected unless `home` is selected."
  - "Init success output now reports the selected layout plus resolved config, data, and state locations instead of a terse repo-only confirmation."
  - "README documentation now leads with the default repo-local XDG path, then explains overrides, doctor inspection, migration prompts, and precedence."
patterns-established:
  - "Repo-local init reports `.gitignore` updates only when the authoritative state-ignore entry actually changed."
  - "Global init writes the selected manifest-backed layout without inventing extra flags or separate command families."
requirements-completed: [REPO-02, CMD-06]
duration: 17m47s
completed: 2026-04-07
---

# Phase 4 Plan 2: Command UX and Migration Help Summary

**Init layout-selection UX, explicit success-path reporting, and default-first operator documentation**

## Performance

- **Duration:** 17m47s
- **Started:** 2026-04-07T03:02:19Z
- **Completed:** 2026-04-07T03:20:06Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments

- Added the approved `changes init` and `changes init global` layout-selection UX with explicit resolved path reporting.
- Kept repo-local `.gitignore` updates authoritative and conditional so the CLI only mentions them when init actually changed the file.
- Rewrote the README around the default XDG workflow, the `home` alternative, `doctor` inspection/migration examples, and concise precedence tables.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add minimal repo/global init layout flags and explicit success-path reporting** - `c1d832d` (test), `41bad06` (feat)
2. **Task 2: Rewrite the README around defaults, alternatives, and migration workflows** - pending final docs commit

## Files Created/Modified

- `internal/config/init_defaults.go` - Added global init selection helpers and authoritative repo-state ignore selection.
- `internal/config/init_defaults_test.go` - Locked global init layout selection defaults and requested-home behavior.
- `internal/app/app.go` - Expanded init result contracts with selected layout, resolved paths, gitignore reporting, and global-init request/result types.
- `internal/app/init.go` - Implemented global init, propagated resolved paths into init results, and tracked whether repo init changed `.gitignore`.
- `internal/app/init_test.go` - Added app-level coverage for repo/global init reporting and conditional gitignore updates.
- `internal/cli/app.go` - Routed `changes init global`, validated `--home`, updated help, and rendered explicit success output.
- `internal/cli/app_integration_test.go` - Added CLI coverage for init help, global-home output, and invalid flag combinations.
- `README.md` - Documented the default-first init and doctor workflows, precedence, and repo-local state ignore behavior.

## Decisions Made

- Kept `init` and `init global` as the only setup entry points and reused the same minimal layout flags for both repo and global scopes.
- Reported resolved config/data/state paths directly from the app-layer init results so CLI rendering stays straightforward and testable.
- Documented the flexible layout model through concrete command examples and short precedence tables instead of proposal-style prose.

## Deviations from Plan

None.

## Issues Encountered

None.

## User Setup Required

None.

## Next Phase Readiness

- Phase 4 now has both the inspection/migration CLI and the init/docs UX needed for rollout verification.
- Ready for phase-level verification and then Phase 5 rollout verification planning.

## Self-Check: PASSED

- Found `.planning/phases/04-command-ux-and-migration-help/04-command-ux-and-migration-help-02-SUMMARY.md`.
- Verified task commits `c1d832d` and `41bad06` exist in `git log`.
