---
phase: 05-rollout-verification
status: ready_for_planning
updated: 2026-04-07
---

# Phase 5 Context: Rollout Verification

## Phase Goal

Make the new layout model the default-safe behavior and verify compatibility across existing and new layouts.

## Locked Scope

Phase 5 is a verification and rollout-confidence phase. It should confirm that the shipped resolver, authority model, init UX, and doctor workflows behave correctly across clean repos, manifest-backed repos, and legacy repos without reintroducing broad legacy-operation compatibility.

This phase should:

- verify that default init behavior is aligned with the approved layout model
- verify that manifest-backed XDG and `home` repos operate correctly without migration
- verify that legacy repos without manifests fail cleanly and remain diagnosable through `doctor`
- verify the focused precedence rules for repo and global init selection
- verify the migration-prompt and authority behavior through a layered matrix of unit, app, CLI, and full-suite checks

This phase should not:

- re-open the manifest-backed-only normal-operation rule
- add a compatibility bridge that makes legacy repos operational again
- redesign command surfaces already locked in earlier phases
- attempt an exhaustive precedence matrix in v1

## Decisions Carried Forward

- Supported styles remain `xdg` and `home`
- Manifest-backed layouts are operationally valid for normal commands
- Legacy layouts remain detectable and diagnosable, but not operationally valid
- `doctor` remains the inspection and migration-brief surface
- `init` and `init global` already expose the approved minimal `--layout` / `--home` grammar
- Repo-local state ignore behavior is already locked and implemented

## Phase 5 Decisions

### Existing Repo Compatibility

Use the mixed compatibility interpretation:

- existing manifest-backed repos must continue to operate without migration
- legacy repos without manifests do not need to operate normally
- Phase 5 must prove that legacy repos fail cleanly and are diagnosable through `doctor`

### Default-Selection Verification Scope

Use the focused precedence set:

- plain `changes init` defaults to repo `xdg`
- plain `changes init global` defaults to global `xdg`
- repo init uses `[repo.init]` global config defaults when present
- `CHANGES_HOME` acts as a style-preference signal for repo init
- `CHANGES_HOME` beats XDG env for global init preference
- explicit flags beat every other source
- invalid `style` / `home` combinations remain rejected

An exhaustive precedence matrix is deferred to future work.

### Verification Depth

Use a layered rollout matrix:

- focused unit tests for selection, precedence, and resolution behavior
- app/service tests for authority and migration interactions
- a small number of CLI integration tests for the most important operator flows
- a full `go test ./...` regression pass

### Legacy Repo Rollout Stance

Legacy repos without manifests are verified and documented, not made operational again in this phase.

## Expected Outputs

Phase 5 planning and execution should produce:

- verification-focused tests for default selection and precedence
- verification-focused tests for manifest-backed repo compatibility
- verification-focused tests for legacy-only failure plus doctor diagnosability
- planning and verification artifacts that make the rollout boundary explicit

## Key Risks

- accidentally weakening the manifest-backed-only operational rule while chasing compatibility
- under-testing precedence interactions between flags, `[repo.init]`, `CHANGES_HOME`, and XDG env inputs
- overbuilding toward an exhaustive matrix instead of the locked focused rollout set

## Read-First References

- `.planning/proposals/layout-resolution.md`
- `.planning/ROADMAP.md`
- `.planning/STATE.md`
- `.planning/REQUIREMENTS.md`
- `.planning/phases/04-command-ux-and-migration-help/04-VERIFICATION.md`
- `internal/config/init_defaults.go`
- `internal/config/resolution.go`
- `internal/app/init.go`
- `internal/app/doctor.go`
- `internal/cli/app.go`
