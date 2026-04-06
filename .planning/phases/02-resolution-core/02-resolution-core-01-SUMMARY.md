---
phase: 02-resolution-core
plan: 01
subsystem: config
tags: [go, resolver, manifest, xdg, home, toml]
requires:
  - phase: 01-layout-proposal
    provides: Approved layout proposal, manifest schema, and phase 2 implementation gate
provides:
  - Shared resolver contracts for global and repo scopes
  - Strict manifest parsing with symbolic preservation and normalized resolved paths
  - Focused tests for bootstrap precedence, supported layouts, manifest safety, and canonical comparison
affects: [02-resolution-core, 03-authority-and-safety, 04-command-ux-and-migration-help]
tech-stack:
  added: []
  patterns: [statusful resolver results, strict TOML manifest validation, nearest-ancestor path canonicalization]
key-files:
  created:
    - internal/config/resolution_types.go
    - internal/config/resolution.go
    - internal/config/resolution_test.go
    - internal/config/manifest.go
    - internal/config/manifest_test.go
  modified:
    - internal/config/resolution.go
key-decisions:
  - "Scope resolution returns both supported style candidates plus preferred and authoritative pointers."
  - "Invalid manifests are classified into structured resolver status instead of surfacing as ordinary-operation errors."
  - "Path equivalence canonicalizes the nearest existing ancestor so symlinked roots compare correctly without rewriting symbolic inputs."
patterns-established:
  - "Resolver Pattern: ResolveAll is the core orchestration entry point and per-scope helpers delegate to it."
  - "Manifest Pattern: Preserve symbolic manifest fields and separate them from normalized resolved filesystem paths."
requirements-completed: [GLBL-01, GLBL-02, REPO-01, MIGR-01]
duration: 10 min
completed: 2026-04-06
---

# Phase 2 Plan 01: Resolution Core Summary

**Shared global and repo resolver core with strict layout manifest validation, symbolic preservation, and normalized candidate evidence**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-06T22:28:44Z
- **Completed:** 2026-04-06T22:39:01Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added an explicit resolver object model for scope, style, status, candidate evidence, manifest data, and resolve options.
- Implemented shared `ResolveAll`, `ResolveGlobal`, and `ResolveRepo` orchestration that preserves both supported style candidates per scope.
- Added strict manifest parsing that preserves symbolic values, expands approved references into normalized paths, rejects repo escapes, and keeps `layout.toml` read-only during resolution.

## Task Commits

Each task was committed atomically:

1. **Task 1: Build the explicit resolver contracts and shared orchestration engine** - `2b2225c` (test), `ac3a37a` (feat)
2. **Task 2: Add strict manifest parsing, symbolic preservation, and normalization safety** - `bf32936` (test), `5a23367` (feat)

## Files Created/Modified
- `internal/config/resolution_types.go` - Exported resolver contracts for scopes, styles, statuses, candidates, manifests, and options.
- `internal/config/resolution.go` - Shared resolver orchestration, supported candidate inspection, bootstrap preference, and scope status summarization.
- `internal/config/resolution_test.go` - TDD coverage for global precedence, supported path shapes, and thin scope wrappers.
- `internal/config/manifest.go` - Strict manifest decode, symbolic expansion, canonical comparison, and repo-local containment checks.
- `internal/config/manifest_test.go` - TDD coverage for symbolic preservation, unsupported keys, escaping paths, and canonical-equivalence handling.

## Decisions Made
- Kept `ResolveAll` as the single orchestration path and made `ResolveGlobal` and `ResolveRepo` thin wrappers over that result.
- Represented ordinary resolver outcomes as structured statuses so malformed or legacy layouts stay diagnosable without turning into boundary errors.
- Canonicalized the nearest existing ancestor during path comparison so symlinked roots and unresolved descendants compare safely and deterministically.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- A transient `.git/index.lock` blocked two commits after overlapping git operations. Retrying those commits sequentially resolved the issue without changing repository content.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/config` now has the resolver core and manifest safety layer needed for compatibility wrapper rewiring in Plan 02-02.
- No code blockers remain for the next plan.

## Self-Check: PASSED
