---
phase: 01-layout-proposal
plan: 01
subsystem: cli
tags: [layout-resolution, xdg, changes-home, doctor]
requires: []
provides:
  - Governing proposal for xdg/home precedence, manifests, authority rules, and repo hygiene
  - Phase-local command contract for init and doctor flows
  - Phase-local diagnostic model for candidate states, JSON output, and migration briefs
affects: [01-layout-proposal, 02-resolution-core, 04-command-ux-and-migration-help]
tech-stack:
  added: []
  patterns: [docs-first contract locking, explicit precedence ordering, tiered doctor output]
key-files:
  created:
    - .planning/phases/01-layout-proposal/01-command-contract.md
    - .planning/phases/01-layout-proposal/01-diagnostic-model.md
  modified:
    - .planning/proposals/layout-resolution.md
key-decisions:
  - "Lock xdg and home as the only supported styles for global and repo scopes."
  - "Keep changes doctor as the only inspection and migration-brief surface, with concise default output and richer explain/json tiers."
patterns-established:
  - "Proposal governs implementation: future phases must reuse the exact init and doctor grammar documented here."
  - "Authority is manifest-backed: legacy layouts are diagnosable but not operationally valid for ordinary commands."
requirements-completed: [CMD-03, CMD-04, CMD-05]
duration: 3 min
completed: 2026-04-06
---

# Phase 1 Plan 1: Layout Proposal Summary

**Locked the authoritative layout proposal plus exact `init` and `doctor` contracts for xdg/home resolution, legacy diagnosis, and migration-safe guidance**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-06T20:17:57Z
- **Completed:** 2026-04-06T20:20:42Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Locked the governing proposal for supported styles, bootstrap precedence, manifest validity, ambiguity rules, and repo hygiene.
- Added a stable command contract for `changes init`, `changes init global`, and tiered `changes doctor` behavior.
- Added a diagnostic model that fixes candidate states, JSON keys, and migration prompt section headings before implementation starts.

## Task Commits

Each task was committed atomically:

1. **Task 1: Finalize the governing layout proposal** - `420c50f` (feat)
2. **Task 2: Write the command and diagnostic contracts** - `d61e0a1` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified

- `.planning/proposals/layout-resolution.md` - Governing proposal for precedence, manifest validity, authority rules, config shape, and migration safety.
- `.planning/phases/01-layout-proposal/01-command-contract.md` - Exact Phase 1 command grammar and validation rules for `init` and `doctor`.
- `.planning/phases/01-layout-proposal/01-diagnostic-model.md` - Candidate-state model, JSON contract, and migration prompt structure.

## Decisions Made

- Locked the precedence wording to `built-in default locations` and made `CHANGES_HOME` advisory only when no conflicting valid on-disk candidate exists.
- Kept manifests structural and explicit: operational validity requires a parsing `layout.toml` whose `scope` and `style` match the candidate.
- Kept migration guidance advisory-only under `changes doctor --migration-prompt`, with explicit single-destination and no-dual-write safety rules.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Manually synced stale planning body text after GSD state updates**
- **Found during:** Final metadata update
- **Issue:** `gsd-tools` advanced the structured fields in `.planning/STATE.md` and marked the completed plan in phase details, but the human-readable progress/status lines in `STATE.md` and the roadmap summary table still showed pre-execution values.
- **Fix:** Updated `.planning/STATE.md` and `.planning/ROADMAP.md` so body text, progress, and next-step status matched the recorded plan completion.
- **Files modified:** `.planning/STATE.md`, `.planning/ROADMAP.md`
- **Verification:** Confirmed Phase 1 shows `1/2` plans complete in the roadmap and `STATE.md` now reports 50% progress with plan 02 as the next target.
- **Committed in:** final docs commit

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** No scope change. The fix kept the planning artifacts internally consistent after execution.

## Issues Encountered

- `gsd-tools` updated structured planning fields but left stale body text in `STATE.md` and the roadmap summary table, so those sections were corrected manually before the final docs commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 1 now has a governing proposal bundle future implementation can reference without reopening command naming or precedence debates.
- Ready for the remaining Phase 1 validation plan and then Phase 2 resolver implementation work.

## Self-Check: PASSED

- Found `.planning/phases/01-layout-proposal/01-01-SUMMARY.md`.
- Verified task commits `420c50f` and `d61e0a1` exist in `git log`.
