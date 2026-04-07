---
phase: 04-command-ux-and-migration-help
verified: 2026-04-07T03:20:03Z
status: passed
score: 8/8 must-haves verified
---

# Phase 4: Command UX and Migration Help Verification Report

**Phase Goal:** Expose the new behavior through clear commands, migration prompt generation, and updated docs
**Verified:** 2026-04-07T03:20:03Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Users can run `changes doctor` in concise, `--explain`, or `--json` modes and inspect global, repo, or all scopes | ✓ VERIFIED | `internal/app/doctor.go` returns one structured doctor result reused by the CLI; covered by `TestDoctorStructuredInspectionSupportsConciseExplainAndJSON`, `TestDoctorDefaultOutputStaysConciseAndExplainAddsDetail`, and `TestDoctorJSONOutputReturnsStructuredInspection`. |
| 2 | Inside a repository, `changes doctor` defaults to repo scope while keeping explicit `--scope global|repo|all` support | ✓ VERIFIED | `internal/cli/app.go` defaults `doctor` to repo scope from repo-root detection; covered by `TestDoctorDefaultsToRepoScopeInsideRepository` and help text assertions in `TestDoctorHelpSurfaceIncludesInspectionAndMigrationFlags`. |
| 3 | Users can generate advisory Markdown migration briefs with deterministic origin and destination facts | ✓ VERIFIED | `internal/app/doctor.go` includes manifest/schema metadata, candidate facts, and inventories in the generated brief; covered by `TestDoctorMigrationPromptIncludesDeterministicMetadata` and `TestDoctorMigrationPromptIncludesConflictNotesAndVerification`. |
| 4 | Ambiguity-facing failures point operators toward the doctor and migration-brief workflow | ✓ VERIFIED | `internal/cli/app.go` extends ambiguous authority failures with terse migration guidance; covered by `TestAmbiguousAuthorityFailurePrintsMigrationHint`. |
| 5 | `changes init` and `changes init global` expose the approved minimal `--layout` and `--home` UX | ✓ VERIFIED | `internal/cli/app.go` documents and validates `changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]` and `changes init global [--layout xdg|home] [--home PATH]`; covered by `TestInitHelpSurfaceIncludesLayoutFlags` and `TestInitRejectsHomeFlagWithoutHomeLayout`. |
| 6 | Successful init output states the selected layout and resolved config, data, and state locations | ✓ VERIFIED | `internal/app/init.go` returns `SelectedLayout`, `ConfigPath`, `DataPath`, and `StatePath`, and `internal/cli/app.go` renders them on success; covered by `TestInitializeReturnsSelectedLayoutAndPaths` and `TestInitGlobalHomeReportsResolvedPaths`. |
| 7 | Repo-local init keeps only the authoritative repo-local state directory ignored and only mentions `.gitignore` when it changed | ✓ VERIFIED | `internal/config/init_defaults.go` selects the authoritative state ignore entry and `internal/app/init.go` tracks whether `.gitignore` changed; covered by `TestInitializeReportsGitignoreChangeOnlyWhenModified`. |
| 8 | README documentation now leads with defaults, then alternatives, then doctor/migration workflows and precedence guidance | ✓ VERIFIED | `README.md` documents the default repo-local XDG path, the `home` alternative, global init, doctor grammar, canonical examples, and repo-local state ignore behavior; verified by the Phase 04 README grep check and manual review. |

**Score:** 8/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/app/doctor.go` | Doctor inspection and migration-prompt application service | ✓ VERIFIED | Exists, substantive, and wired through the CLI. |
| `internal/cli/app.go` | `doctor`, `init`, and `init global` CLI routing, help, and output rendering | ✓ VERIFIED | Exists, substantive, and wired; help and success/error output match the locked command contract. |
| `internal/app/init.go` | Repo/global init orchestration plus result metadata for selected layout and paths | ✓ VERIFIED | Exists, substantive, and wired; repo and global init flows share the approved selection grammar. |
| `internal/config/init_defaults.go` | Deterministic repo/global init selection helpers and authoritative state-ignore selection | ✓ VERIFIED | Exists, substantive, and wired into init orchestration and tests. |
| `README.md` | Default-first operator documentation for init, doctor, migration prompts, precedence, and repo-local state ignores | ✓ VERIFIED | Exists and includes the approved command surfaces plus the canonical Phase 4 examples. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| README command/documentation coverage | `rg -n 'changes init \[--layout xdg\|home\] \[--home PATH\]|changes init global \[--layout xdg\|home\] \[--home PATH\]|changes doctor \[--scope global\|repo\|all\] \[--explain\] \[--json\]|changes doctor --migration-prompt --scope global\|repo --to xdg\|home \[--home PATH\] \[--output PATH\]|changes doctor --scope repo --explain|changes doctor --scope global|changes doctor --migration-prompt --scope repo --to home|\.local/state|\.changes/state' README.md` | Matching lines found for both init forms, both doctor forms, canonical examples, and both repo-local state directories | ✓ PASS |
| Phase 4 package-level regression coverage | `go test ./internal/config ./internal/app ./internal/cli -count=1` | `ok` for all three packages | ✓ PASS |
| Repository-wide regression coverage | `go test ./...` | All packages passed | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `GLBL-03` | `04-01` | `changes` can explain which global layout source won and why | ✓ SATISFIED | `doctor` can inspect global scope in concise, explain, and JSON modes, and the CLI defaults/help expose the approved surfaces. |
| `REPO-02` | `04-02` | `changes init` can choose the repo-local layout style using a clean, documented command shape | ✓ SATISFIED | `changes init` now accepts the approved `--layout` and optional `--home` flags, with README and help text in sync. |
| `AUTH-03` | `04-01` | Ambiguity errors can direct the user to generate an LLM prompt to help merge or migrate to one authoritative location | ✓ SATISFIED | Ambiguous authority failures now point to `changes doctor --migration-prompt ...`. |
| `MIGR-02` | `04-01` | `changes` can generate an LLM prompt for migrating between supported layouts using deterministically gathered source and destination details | ✓ SATISFIED | `doctor --migration-prompt` emits deterministic Markdown to stdout or `--output PATH`. |
| `MIGR-03` | `04-01` | Migration assistance includes origin metadata, destination metadata, and any detected ambiguity or conflict signals | ✓ SATISFIED | Migration briefs enumerate source/destination facts, manifest/schema data, inventories, and conflict notes. |
| `CMD-06` | `04-02` | Repo-local initialization updates `.gitignore` so the authoritative repo-local `state` directory is ignored consistently | ✓ SATISFIED | Repo `xdg` and repo `home` both select one authoritative ignore entry, and init reports `.gitignore` changes only when they occur. |

Orphaned phase requirements: none. `REQUIREMENTS.md` maps only `GLBL-03`, `REPO-02`, `AUTH-03`, `MIGR-02`, `MIGR-03`, and `CMD-06` to Phase 4, and all six are satisfied.

### Gaps Summary

No blocking gaps were found. Phase 4 achieves the roadmap goal: operators can inspect and explain layout resolution, generate deterministic migration briefs, choose repo/global init layouts on the approved command shapes, and follow documentation that now reflects the default-first model and canonical workflows.

---

_Verified: 2026-04-07T03:20:03Z_  
_Verifier: Claude (gsd-verifier)_
