---
phase: 06-legacy-layout-repair
verified: 2026-04-07T08:01:06Z
status: passed
score: 5/5 must-haves verified
---

# Phase 6: Legacy Layout Repair Report

**Phase Goal:** Add an operator-friendly repo-local repair flow for legacy layouts while preserving authority and single-target safety guarantees  
**Verified:** 2026-04-07T08:01:06Z  
**Status:** passed  
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Operators can repair a legacy repo-local layout with `changes doctor --scope repo --repair` instead of hand-authoring `layout.toml` | ✓ VERIFIED | `TestDoctorRepairRepoStampsPreferredLegacyCandidate` passes in [internal/app/doctor_repair_test.go](/Users/tim.shadel/projects/codefactory/changes/internal/app/doctor_repair_test.go), and the CLI success path is covered by `TestDoctorRepairRepairsLegacyRepoAndReportsAuthoritativeLayout` in [internal/cli/doctor_repair_integration_test.go](/Users/tim.shadel/projects/codefactory/changes/internal/cli/doctor_repair_integration_test.go). |
| 2 | Repair fails loudly when more than one repo-local legacy candidate could be repaired | ✓ VERIFIED | `TestDoctorRepairRepoRejectsAmbiguousLegacyCandidates` and `TestDoctorRepairFailsLoudlyOnAmbiguousLegacyRepo` prove the app and CLI both refuse ambiguous repair candidates. |
| 3 | A successful repair leaves ordinary commands operational against the repaired authoritative layout | ✓ VERIFIED | `TestDoctorRepairRepoLeavesOrdinaryCommandsOperational` and `TestDoctorRepairLeavesStatusOperational` prove post-repair `LoadWithAuthority` and `changes status` recovery. |
| 4 | Repair stamps metadata only, preserves the authoritative state ignore rule, and does not migrate or dual-write data | ✓ VERIFIED | `TestDoctorRepairRepoDoesNotMigrateData`, `TestDoctorRepairRepoPreservesAuthoritativeStateIgnoreRule`, and the shared manifest parity checks in [internal/config/manifest_repair_test.go](/Users/tim.shadel/projects/codefactory/changes/internal/config/manifest_repair_test.go) all pass. |
| 5 | Operator docs clearly distinguish repair from migration guidance | ✓ VERIFIED | [README.md](/Users/tim.shadel/projects/codefactory/changes/README.md) now explains when repair is appropriate and when operators should use `doctor --migration-prompt` instead. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| [internal/app/doctor.go](/Users/tim.shadel/projects/codefactory/changes/internal/app/doctor.go) | Repo-local repair orchestration and post-repair validation | ✓ VERIFIED | Adds repo-only repair flow, candidate selection, manifest stamping, `.gitignore` preservation, and authority re-checking. |
| [internal/config/manifest.go](/Users/tim.shadel/projects/codefactory/changes/internal/config/manifest.go) | Shared repo manifest writer reused by init and repair | ✓ VERIFIED | Centralizes repo manifest bytes so init and repair stamp the same symbolic manifest contract. |
| [internal/cli/app.go](/Users/tim.shadel/projects/codefactory/changes/internal/cli/app.go) | `doctor --repair` CLI parsing, validation, and messaging | ✓ VERIFIED | Restricts repair to repo scope and rejects incompatible inspection/migration flags. |
| [internal/app/doctor_repair_test.go](/Users/tim.shadel/projects/codefactory/changes/internal/app/doctor_repair_test.go) | App-layer repair coverage | ✓ VERIFIED | Covers stamping, ambiguity refusal, post-repair operability, no-migration, and `.gitignore` preservation. |
| [internal/cli/doctor_repair_integration_test.go](/Users/tim.shadel/projects/codefactory/changes/internal/cli/doctor_repair_integration_test.go) | Operator-facing repair integration coverage | ✓ VERIFIED | Covers help surface, success/failure output, invalid flag combinations, and post-repair `status` behavior. |
| [README.md](/Users/tim.shadel/projects/codefactory/changes/README.md) | Repair-versus-migration operator guidance | ✓ VERIFIED | Documents the narrow repair workflow and when migration prompts are still required. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Focused app/config repair behaviors | `go test ./internal/config ./internal/app -run 'TestDoctorRepairRepoStampsPreferredLegacyCandidate|TestDoctorRepairRepoRejectsAmbiguousLegacyCandidates|TestDoctorRepairRepoLeavesOrdinaryCommandsOperational|TestDoctorRepairRepoDoesNotMigrateData|TestDoctorRepairRepoPreservesAuthoritativeStateIgnoreRule|TestRepoLayoutManifestWriterMatchesInitSymbolicForms' -count=1` | Passed | ✓ PASS |
| CLI repair workflow and repo-only validation | `go test ./internal/cli -run 'TestDoctorRepairHelpSurfaceIncludesRepoOnlyGrammar|TestDoctorRepairRepairsLegacyRepoAndReportsAuthoritativeLayout|TestDoctorRepairFailsLoudlyOnAmbiguousLegacyRepo|TestDoctorRepairLeavesStatusOperational|TestDoctorRepairRejectsInvalidFlagCombinations' -count=1` | Passed | ✓ PASS |
| Package-level regression | `go test ./internal/config ./internal/app ./internal/cli -count=1` | Passed | ✓ PASS |
| Repository-wide regression | `go test ./...` | Passed | ✓ PASS |

### Gaps Summary

No blocking gaps were found. The repo-local legacy repair workflow now closes the shipped operator gap without reopening migration semantics: repair is explicit, repo-only, fail-loud on ambiguity, and leaves ordinary commands operational once the authoritative manifest is restored.

---

_Verified: 2026-04-07T08:01:06Z_  
_Verifier: Codex execute-phase closeout_
