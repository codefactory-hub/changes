---
phase: 06-legacy-layout-repair
plan: 01
subsystem: cli
tags: [go, cli, doctor, layout, repair, manifest]
requires:
  - phase: 03-authority-and-safety
    provides: authoritative layout checks and warning contracts
  - phase: 05-rollout-verification
    provides: legacy layout boundary and manifest-backed compatibility coverage
provides:
  - repo-local `doctor --repair` recovery flow for legacy layouts
  - shared repo manifest writer reused by init and repair
  - CLI repair validation, reporting, and operator guidance
affects: [doctor, init, legacy-layouts, documentation]
tech-stack:
  added: []
  patterns: [transactional repo metadata repair, shared manifest encoding, fail-loud legacy selection]
key-files:
  created:
    - internal/app/doctor_repair_test.go
    - internal/config/manifest_repair_test.go
    - internal/cli/doctor_repair_integration_test.go
  modified:
    - internal/app/app.go
    - internal/app/doctor.go
    - internal/app/init.go
    - internal/config/manifest.go
    - internal/cli/app.go
    - README.md
key-decisions:
  - "Repair selects from the full repo candidate set instead of resolver Preferred so home-only legacy repos can be repaired safely."
  - "Repair is allowed only when exactly one legacy repo-local candidate exists and no manifest-backed authority already exists."
  - "Repo manifest bytes now come from internal/config so init and repair stamp the same symbolic layout contract."
patterns-established:
  - "Doctor repair mutates only layout.toml and the authoritative repo-local .gitignore entry, then revalidates authority before reporting success."
  - "CLI repair stays under doctor and rejects inspection or migration flags that would blur the workflow."
requirements-completed: [REPAIR-01, REPAIR-02, REPAIR-03, REPAIR-04, SAFE-01, SAFE-02, SAFE-03]
duration: 6m
completed: 2026-04-07
---

# Phase 6 Plan 01: Legacy Layout Repair Summary

**Repo-local doctor repair that restores one authoritative legacy manifest, preserves repo state-ignore hygiene, and leaves ordinary commands operational**

## Performance

- **Duration:** 6 min
- **Started:** 2026-04-07T07:54:59Z
- **Completed:** 2026-04-07T08:01:06Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments

- Added repo-local repair orchestration to `Doctor`, including fail-loud candidate selection, manifest stamping, `.gitignore` preservation, and post-repair authority validation.
- Moved repo manifest encoding into `internal/config` so init and repair stamp the same symbolic `layout.toml` contract.
- Exposed `changes doctor --scope repo --repair` in the CLI with narrow validation, concise repair reporting, and README guidance for repair versus migration prompts.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add repo-local repair orchestration and shared manifest stamping** - `8e7ec8b` (test), `97c9469` (feat)
2. **Task 2: Wire `changes doctor --scope repo --repair` through the CLI and operator docs** - `1d28b98` (test), `cb25eeb` (feat)

_Note: Both tasks used TDD red/green commits._

## Files Created/Modified

- `internal/app/doctor_repair_test.go` - app-layer repair behavior coverage for stamping, ambiguity refusal, operational recovery, no-migration, and `.gitignore` preservation.
- `internal/config/manifest.go` - shared repo layout manifest writer used by init and repair.
- `internal/config/manifest_repair_test.go` - symbolic manifest parity tests for xdg and home repo layouts.
- `internal/app/app.go` - doctor request/result contract extended with repair input and structured repair output.
- `internal/app/doctor.go` - repo-only repair orchestration, candidate selection, manifest stamping, and post-repair validation.
- `internal/app/init.go` - repo init now reuses the shared manifest writer.
- `internal/cli/doctor_repair_integration_test.go` - operator-facing repair help, validation, success, ambiguity, and post-repair status coverage.
- `internal/cli/app.go` - CLI `--repair` flag parsing, validation, and repair summary rendering.
- `README.md` - operator guidance for when repair is appropriate versus when migration prompts are required.

## Decisions Made

- Repair chooses the sole legacy candidate directly from the resolver candidate set instead of `resolution.Preferred`, because repo inspection defaults still prefer `xdg` even when the only repairable layout is `home`.
- Repair refuses to run when any manifest-backed repo authority already exists, preventing a second authoritative repo-local layout from being stamped accidentally.
- Repair remains a narrow `doctor` subflow rather than a new command family, matching the shipped inspection-and-recovery contract.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Legacy repo-local layouts with exactly one repairable candidate can now be recovered without hand-authoring `layout.toml`.
- Follow-on migration validation work can build on the new repaired-versus-migrate operator boundary without revisiting authority rules.

## Self-Check: PASSED

- Summary file created at `.planning/phases/06-legacy-layout-repair/06-01-SUMMARY.md`
- Task commits verified: `8e7ec8b`, `97c9469`, `1d28b98`, `cb25eeb`
