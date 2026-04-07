---
phase: 03-authority-and-safety
verified: 2026-04-07T01:59:37Z
status: passed
score: 9/9 must-haves verified
---

# Phase 3: Authority and Safety Verification Report

**Phase Goal:** Enforce authoritative-layout rules and make ambiguity visible, inspectable, and non-destructive
**Verified:** 2026-04-07T01:59:37Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Commands fail loudly when multiple supported layouts compete for authority | ✓ VERIFIED | `internal/config/resolution.go` groups operational authorities and returns `StatusAmbiguous` when more than one distinct authority group exists; `internal/config/authority.go` turns that into a typed `AuthorityError`; covered by `TestResolveRepoAmbiguousWhenDistinctOperationalCandidatesCompete`, `TestCheckScopeAuthorityReturnsAmbiguousError`, and `TestAppCreateFailsWithTerseAmbiguousDoctorHint`. |
| 2 | Errors explain the competing candidates and suggest how the operator can choose one | ✓ VERIFIED | `AuthorityError.Error()` includes scope, competing style names, and `changes doctor --scope ...` guidance in `internal/config/authority.go`; covered by `TestAuthorityErrorMessageIncludesDoctorHint` and CLI integration tests for repo/global ambiguity. |
| 3 | Normal command execution still writes to exactly one authoritative target | ✓ VERIFIED | `Initialize` derives all write paths from one `selection` object and writes only those paths in `internal/app/init.go`; `CommitRelease` and `runCreate` recheck `RequireRepoWriteAuthority` before writing in `internal/app/app.go` and `internal/cli/create.go`; full suite passed with `go test ./...`. |
| 4 | A single operationally valid layout remains authoritative even when invalid-manifest or legacy-only siblings also exist | ✓ VERIFIED | `resolveScopeFromCandidates` keeps `StatusResolved`, assigns one `Authoritative` candidate, and records sibling warnings in `internal/config/resolution.go`; covered by resolver, config, app, and CLI tests for legacy/invalid siblings. |
| 5 | Equivalent candidates that canonicalize to the same physical location collapse to one authority target | ✓ VERIFIED | `candidateAuthorityKey` canonicalizes the resolved root through `canonicalPathForComparison` in `internal/config/resolution.go`; covered by `TestResolveRepoCollapsesEquivalentOperationalCandidates`. |
| 6 | Unsupported schema versions are operationally invalid for ordinary commands | ✓ VERIFIED | `loadLayoutManifest` rejects any `schema_version` other than `1` in `internal/config/manifest.go`; covered by `TestResolveManifestRejectsUnsupportedSchemaVersion`. |
| 7 | Ordinary commands print concise stderr warnings when one authoritative layout exists alongside legacy-only or invalid siblings | ✓ VERIFIED | `printAuthorityWarnings` formats CLI-only warning lines in `internal/cli/app.go`; `Status`, `PlanRelease`, `Render`, `init`, and `create` all propagate structured warnings; covered by CLI integration tests for status/create/release/render profiles/init. |
| 8 | `changes init` fails loudly if the global layout authority used for repo-init defaults is ambiguous, legacy-only, or invalid-manifest | ✓ VERIFIED | `loadGlobalRepoInitDefaults` resolves global authority through `ResolveGlobal` + `CheckScopeAuthority` in `internal/app/init.go`; ambiguous global authority is covered by `TestInitializeRejectsAmbiguousGlobalDefaultsAuthority` and `TestAppInitFailsWithTerseAmbiguousGlobalDoctorHint`. |
| 9 | Write-capable commands explicitly refuse to write when repo authority is ambiguous, legacy-only, invalid-manifest, or uninitialized | ✓ VERIFIED | `RequireRepoWriteAuthority` is the shared write gate in `internal/config/config.go`; `CommitRelease` calls it immediately before `releases.Write`, `runCreate` calls it before `fragments.Create`, and `init` rejects non-uninitialized authority failures before any writes begin. |

**Score:** 9/9 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/config/resolution.go` | Authority grouping, sibling-warning collection, authoritative candidate selection | ✓ VERIFIED | Exists, substantive, and wired; `gsd-tools verify artifacts` passed; focused config tests passed. |
| `internal/config/manifest.go` | Strict schema-version validation and manifest-backed operational validity checks | ✓ VERIFIED | Exists, substantive, and wired; rejects unsupported schema versions without mutation. |
| `internal/config/authority.go` | Shared structured warning and terse authority-error helpers | ✓ VERIFIED | Exists, substantive, and wired; exported surface matches plan contract. |
| `internal/config/config.go` | Warning-aware config loading and explicit repo write-authority helpers | ✓ VERIFIED | Exists, substantive, and wired; `LoadWithAuthority` and `RequireRepoWriteAuthority` are used by app and CLI layers. |
| `internal/app/app.go` | Authority-warning propagation for status/release/render flows | ✓ VERIFIED | Exists, substantive, and wired; result structs carry `AuthorityWarnings`, write path rechecks authority. |
| `internal/app/init.go` | Explicit init-time repo authority enforcement and authority-checked global-default lookup | ✓ VERIFIED | Exists, substantive, and wired; no special authority exception for `init`. |
| `internal/cli/app.go` | Stderr warning rendering and terse authority failure surfacing | ✓ VERIFIED | Exists, substantive, and wired; warnings stay in CLI presentation layer. |
| `internal/cli/create.go` | Explicit create-path write gate before fragment creation | ✓ VERIFIED | Exists, substantive, and wired; warning-aware load plus explicit repo write gate. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `internal/config/resolution.go` | `internal/config/manifest.go` | candidate inspection, schema-version gating, canonical-equivalence collapse | ✓ WIRED | `gsd-tools verify key-links` passed; `loadLayoutManifest`, `equivalentPaths`, and schema checks are present. |
| `internal/config/authority.go` | `internal/config/resolution_types.go` | shared `ScopeResolution`, `Candidate`, and warning contracts | ✓ WIRED | `gsd-tools` verified the shared contracts are referenced directly. |
| `internal/config/config.go` | `internal/config/authority.go` | `LoadWithAuthority` and `RequireRepoWriteAuthority` | ✓ WIRED | `resolveRepoAuthority` calls `ResolveRepo` then `CheckScopeAuthority`. |
| `internal/app/app.go` | `internal/config/config.go` | warning-aware config loading and explicit write gates | ✓ WIRED | `Status`, `PlanRelease`, and `Render` call `LoadWithAuthority`; `CommitRelease` calls `RequireRepoWriteAuthority`. |
| `internal/app/init.go` | `internal/config/authority.go` | global-layout authority check before repo-init defaults are consumed | ✓ WIRED | `selectInitializeLayout` and `loadGlobalRepoInitDefaults` both call `CheckScopeAuthority`. |
| `internal/cli/app.go` | `internal/app/app.go` | stderr warning rendering before normal stdout output | ✓ WIRED | `runInit`, `runStatus`, `runRelease`, and `runRender` print warnings from app-layer results before stdout output. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `internal/config/config.go` | `check.Authoritative`, `check.Warnings` | `resolveRepoAuthority` → `ResolveRepo` → candidate inspection + manifest parsing | Yes — authority data is derived from on-disk manifests/config artifacts, not hardcoded fallbacks | ✓ FLOWING |
| `internal/app/app.go` | `AuthorityWarnings` on `StatusResult`, `ReleasePlan`, `RenderResult` | `config.LoadWithAuthority` and `config.RequireRepoWriteAuthority` | Yes — warnings and gates flow from real repo layout resolution | ✓ FLOWING |
| `internal/app/init.go` | `authorityWarnings`, global repo-init defaults | `selectInitializeLayout` → `loadGlobalRepoInitDefaults` → `ResolveGlobal` + `CheckScopeAuthority` + global `config.toml` read | Yes — global defaults and warnings come from actual resolved global layout state | ✓ FLOWING |
| `internal/cli/app.go` / `internal/cli/create.go` | stderr warning lines and fail-loud errors | app/config authority results passed into CLI renderers | Yes — CLI output is driven by structured warning/error values from lower layers | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Authority core behaviors execute as implemented | `go test ./internal/config -run 'TestResolveRepoAmbiguousWhenDistinctOperationalCandidatesCompete|TestResolveRepoAllowsSingleOperationalCandidateWithLegacySiblingWarning|TestResolveRepoAllowsSingleOperationalCandidateWithInvalidSiblingWarning|TestResolveRepoCollapsesEquivalentOperationalCandidates|TestResolveManifestRejectsUnsupportedSchemaVersion|TestCheckScopeAuthorityReturnsAmbiguousError|TestCheckScopeAuthorityReturnsLegacyOnlyError|TestCheckScopeAuthorityReturnsInvalidManifestError|TestCheckScopeAuthorityReturnsStructuredWarnings|TestAuthorityErrorMessageIncludesDoctorHint|TestLoadWithAuthorityReturnsWarningForLegacySibling|TestLoadWithAuthorityReturnsWarningForInvalidSibling|TestLoadWithAuthorityReturnsAmbiguousAuthorityError|TestRequireRepoWriteAuthorityRejectsLegacyOnlyRepo|TestRequireRepoWriteAuthorityRejectsAmbiguousRepo' -count=1` | `ok github.com/example/changes/internal/config 0.452s` | ✓ PASS |
| App-layer warning propagation and init/release gates execute as implemented | `go test ./internal/app -run 'TestStatusCarriesAuthorityWarningsForLegacySibling|TestPlanReleaseCarriesAuthorityWarningsForLegacySibling|TestRenderCarriesAuthorityWarningsForInvalidSibling|TestInitializeRejectsAmbiguousRepoAuthorityBeforeWriting|TestInitializeRejectsAmbiguousGlobalDefaultsAuthority|TestInitializeCarriesWarningsForGlobalDefaultsSibling|TestCommitReleaseRejectsAmbiguousRepoAuthorityBeforeWriting' -count=1` | `ok github.com/example/changes/internal/app 0.981s` | ✓ PASS |
| CLI warning rendering and terse ambiguity failures execute as implemented | `go test ./internal/cli -run 'TestAppStatusPrintsAuthorityWarningToStderr|TestAppCreatePrintsAuthorityWarningToStderr|TestAppReleasePrintsAuthorityWarningToStderr|TestAppRenderProfilesPrintsAuthorityWarningToStderr|TestAppCreateFailsWithTerseAmbiguousDoctorHint|TestAppInitPrintsGlobalAuthorityWarningToStderr|TestAppInitFailsWithTerseAmbiguousGlobalDoctorHint' -count=1` | `ok github.com/example/changes/internal/cli 1.375s` | ✓ PASS |
| Phase work does not regress the repository | `go test ./...` | All packages passed | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `AUTH-01` | `03-01`, `03-02` | If multiple supported global or repo-local layouts exist at once, `changes` fails instead of choosing silently | ✓ SATISFIED | Resolver returns `StatusAmbiguous`, authority checks return typed failures, and CLI/app tests cover repo/global ambiguity failures. |
| `AUTH-02` | `03-01`, `03-02` | Ambiguity errors identify the competing authoritative candidates and suggest how to choose between them | ✓ SATISFIED | `AuthorityError.Error()` names competing styles and points to `changes doctor --scope ...`; CLI tests verify repo/global doctor hints. |
| `MIGR-04` | `03-02` | `changes` never dual-writes to origin and destination layouts during normal operation | ✓ SATISFIED | `Initialize` writes only selected layout paths, and `create`/`release` explicitly recheck repo authority immediately before writes. |

Orphaned phase requirements: none. `REQUIREMENTS.md` maps only `AUTH-01`, `AUTH-02`, and `MIGR-04` to Phase 3, and all three appear in plan frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| None | - | No TODO/FIXME/placeholder markers or user-visible stub implementations found in the Phase 03 implementation files | ℹ️ Info | No blocker or warning-level anti-patterns detected in scanned files |

### Human Verification Required

None. This phase is code-path and CLI behavior work with direct automated coverage for the fail-loud and warning surfaces that matter here.

### Gaps Summary

None. Phase 03 achieves the roadmap goal in code: authority selection is explicit and fail-loud, competing candidates are surfaced with operator guidance, mixed sibling states stay visible as warnings, and managed write paths recheck authority before mutating disk.

---

_Verified: 2026-04-07T01:59:37Z_  
_Verifier: Claude (gsd-verifier)_
