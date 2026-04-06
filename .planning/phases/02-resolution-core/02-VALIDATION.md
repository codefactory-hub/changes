---
phase: 2
slug: 02-resolution-core
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-04-06
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package |
| **Config file** | none — command-driven test execution |
| **Quick run command** | `go test ./internal/config -count=1` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/config -count=1`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | GLBL-01 | T-02-03 / T-02-04 | Resolver preserves candidate evidence and reports status without silently choosing a layout | unit | `go test ./internal/config -run 'TestResolveGlobalPrefersChangesHomeOverXDG|TestResolveGlobalPathsForSupportedStyles|TestResolveRepoPathsForSupportedStyles|TestResolveAllReturnsThinScopeWrappers' -count=1` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | MIGR-01 | T-02-01 / T-02-02 | Manifest parsing rejects invalid input, preserves symbolic metadata, and stays read-only during ordinary resolution | unit | `go test ./internal/config -run 'TestResolveManifestPreservesSymbolicLayoutWithoutRewrite|TestResolveManifestRejectsUnsupportedKeys|TestResolveRepoManifestRejectsEscapingPaths|TestResolveCanonicalizesEquivalentRootsForComparison' -count=1` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 2 | GLBL-02 / REPO-01 | T-02-05 / T-02-06 | Compatibility helpers and config loading delegate to resolver-backed authoritative paths | unit | `go test ./internal/config -run 'TestLoadUsesResolverBackedRepoConfigPath|TestPathHelpersUseResolverAuthoritativePaths|TestLoadReturnsInitHintForUninitializedRepoLayout' -count=1` | ❌ W0 | ⬜ pending |
| 02-02-02 | 02 | 2 | REPO-03 | T-02-07 / T-02-08 | Repo init selection follows the locked precedence table and exposes the chosen state ignore path | unit | `go test ./internal/config -run 'TestSelectRepoInitLayoutDefaultsToXDG|TestSelectRepoInitLayoutUsesRepoInitHomeDefault|TestSelectRepoInitLayoutPrefersChangesHomeSignalOverXDGSignal' -count=1` | ❌ W0 | ⬜ pending |
| 02-02-03 | 02 | 2 | REPO-03 | T-02-07 / T-02-08 | Current init flow consumes the shared repo-init selection helper instead of hard-coded layout joins | service | `go test ./internal/app -run 'TestInitializeUsesSelectedRepoLayoutDefaults|TestInitializeHomeLayoutAddsStateGitignore' -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/config/resolution_test.go` — precedence, supported-style path resolution, status, and containment coverage for `GLBL-01`, `GLBL-02`, and `REPO-01`
- [ ] `internal/config/manifest_test.go` — strict decode, symbolic preservation, and no-rewrite coverage for `MIGR-01`
- [ ] `internal/config/config_test.go` — resolver-backed compatibility helper and config loading regression coverage
- [ ] `internal/config/init_defaults_test.go` — deterministic repo-init precedence coverage for `REPO-03`
- [ ] `internal/app/app_test.go` — init-path coverage proving the shared selection helper is consumed by the real bootstrap flow

---

## Manual-Only Verifications

All phase behaviors have automated verification.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
