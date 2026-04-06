# Codebase Concerns

**Analysis Date:** 2026-04-06

The current Go test suite passes under `go test ./...`, so the concerns below focus on structural risk, operational failure modes, and areas that are likely to degrade as the project grows rather than on an actively failing build.

## Tech Debt

**Command orchestration is concentrated in a few large files:**
- Issue: Core behavior is packed into a small number of multi-purpose files instead of command- or domain-scoped units. `internal/app/app.go` owns status, release planning, commit-time staleness checks, render selection, and shared state loading. `internal/app/init.go` combines initialization orchestration, bootstrap history generation, prompt generation, gitignore mutation, and rollback bookkeeping. `internal/cli/app.go` mixes command routing, help text, and command-specific argument handling. `internal/releases/releases.go` carries persistence, validation, lineage, sorting, and selection logic.
- Files: `internal/app/app.go`, `internal/app/init.go`, `internal/cli/app.go`, `internal/releases/releases.go`
- Impact: Small feature work crosses multiple responsibilities at once, raises regression risk, and makes it harder to isolate bugs. These files are already among the largest project-owned Go sources.
- Fix approach: Split by command and lifecycle step. Keep CLI parsing in `internal/cli/*`, command orchestration in command-specific app services, and release-record querying/validation in smaller units with narrower APIs.

**Production release safety checks are split across workflows:**
- Issue: The dry-run workflow validates GoReleaser config with `./scripts/verify-release-config.sh`, but the production tag workflow does not. The production workflow runs `./scripts/prepare-release-notes.sh` and then publishes directly with GoReleaser.
- Files: `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, `scripts/verify-release-config.sh`
- Impact: A release can pass local tests and even the production workflow setup while still skipping the extra config validation path that the dry-run workflow treats as required.
- Fix approach: Share a single verified release pipeline between `.github/workflows/release.yml` and `.github/workflows/release-dry-run.yml`, or make the production workflow call `./scripts/verify-release-config.sh` before publish.

## Known Bugs

**`changes create --edit` fails with common multi-word editor commands:**
- Symptoms: `changes create --edit` launches the value of `$VISUAL` or `$EDITOR` as a single executable path. Values like `code --wait`, `vim -f`, or `nvim -f` are treated as binary names instead of command plus flags.
- Files: `internal/cli/create.go`
- Trigger: Set `$VISUAL` or `$EDITOR` to a command string that includes arguments, then run `changes create --edit`.
- Workaround: Point `$VISUAL` or `$EDITOR` at a wrapper script or a bare executable name with no inline flags.

**Release notes generation silently degrades to placeholder content in the publish path:**
- Symptoms: `scripts/prepare-release-notes.sh` writes placeholder release notes if `.config/changes/config.toml` is missing or if `go run ./cmd/changes render --latest ...` fails. The production workflow still continues into GoReleaser with `.dist/release-notes.md`.
- Files: `scripts/prepare-release-notes.sh`, `.github/workflows/release.yml`
- Trigger: Missing repo-local config, invalid render configuration, missing release records, or render/runtime failures during the release workflow.
- Workaround: Run `.github/workflows/release-dry-run.yml` or the local scripts from `docs/releasing/RELEASING.md` before tagging a release, and inspect `.dist/release-notes.md` explicitly.

## Security Considerations

**Release publication continues after content-generation failure:**
- Risk: The production release workflow accepts placeholder release notes as a valid fallback. This is not a secret-exposure bug, but it is a supply-chain and operational trust risk because public release artifacts can be published with incomplete or misleading release metadata.
- Files: `scripts/prepare-release-notes.sh`, `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`
- Current mitigation: A separate dry-run workflow performs `./scripts/verify-release-config.sh` and snapshot validation before publishing.
- Recommendations: Fail the production workflow when generated release notes fall back to placeholder text, and run the same verification script in both dry-run and production workflows.

## Performance Bottlenecks

**Every high-level command reparses the full fragment and release state:**
- Problem: `Status`, `PlanRelease`, `Render`, and `CommitRelease` all call `loadState`, which reparses every fragment and every release record from disk. `changes release` effectively does this twice across planning and commit.
- Files: `internal/app/app.go`, `internal/fragments/fragments.go`, `internal/releases/releases.go`
- Cause: `loadState` delegates to `fragments.List` and `releases.List` on each call, and `releases.List` walks the whole releases tree and validates the full record set every time.
- Improvement path: Introduce a cached state/index layer for one command invocation, or persist compact indexes for fragments and release heads instead of full rescan-and-validate on each operation.

**Release-chain rendering does repeated whole-document renders under tight size limits:**
- Problem: Chain rendering retries by trimming one bundle at a time and rerendering the entire document until it fits `max_chars`.
- Files: `internal/render/render.go`
- Cause: `renderChain` decrements `keep` in a loop and calls `renderDocument` on every iteration.
- Improvement path: Precompute per-bundle rendered sizes or use a binary search / cumulative-length strategy instead of rerendering the full prefix on every retry.

## Fragile Areas

**Version parsing failures still crash the process in core release selection code:**
- Files: `internal/versioning/versioning.go`, `internal/releases/releases.go`
- Why fragile: `versioning.MustParse` and `versioning.Compare` panic on invalid versions, and those panic-based helpers are used from sorting and release-head selection paths. Most file-backed inputs are validated first, but any future caller that passes partially validated data can crash the CLI instead of returning a normal error.
- Safe modification: Replace panic-based helpers in selection/sorting code with error-returning parsing at the edges, then keep panic-free comparison inside the release package.
- Test coverage: `internal/versioning/versioning_test.go` covers parsing and comparison behavior, but there is no panic-guard coverage around `internal/releases/releases.go` call sites.

**Initialization rollback is best-effort and relies on filesystem details:**
- Files: `internal/app/init.go`
- Why fragile: `Initialize` mutates `.config`, `CHANGELOG.md`, `.gitignore`, `.local/share/changes/...`, and optional bootstrap artifacts in one transaction object. Rollback restores files by rewriting backups and suppresses directory-removal failures with a string match on `"directory not empty"`.
- Safe modification: Keep initialization changes narrow, preserve rollback behavior with integration tests before refactors, and prefer structured filesystem error handling over string matching.
- Test coverage: `internal/app/app_test.go` and `internal/cli/app_integration_test.go` cover initialization flows, but they do not exercise many partial-failure rollback branches.

## Scaling Limits

**Repository-size growth directly increases command latency:**
- Current capacity: Each `changes status`, `changes render`, and `changes release` planning step performs one full fragment scan plus one full release-record scan. `changes release` commit validation performs another full scan before writing.
- Limit: Latency scales with the number of files under `config.FragmentsDir(repoRoot, cfg)` and `config.ReleasesDir(repoRoot, cfg)`, because there is no persistent index, incremental update path, or cached read model.
- Scaling path: Add an invocation-scoped cache first, then consider persisted indexes for release heads, fragment reachability, and release lineage.

## Dependencies at Risk

**Not detected:**
- Risk: The dependency set is small and stable (`github.com/BurntSushi/toml` and `github.com/Masterminds/semver/v3` in `go.mod`).
- Impact: No immediate package-level migration pressure is evident from the current codebase.
- Migration plan: Not applicable.

## Missing Critical Features

**No strict production gate for release-note quality:**
- Problem: Publishing tolerates placeholder release notes instead of enforcing successful render output.
- Blocks: Reliable release metadata quality in `.github/workflows/release.yml` and automated confidence that a tagged release describes the actual contents.

## Test Coverage Gaps

**Release shell scripts and production workflow behavior are not exercised by the main CI job:**
- What's not tested: `scripts/prepare-release-notes.sh`, `scripts/verify-release-config.sh`, and the workflow split between `.github/workflows/ci.yml` and `.github/workflows/release.yml`.
- Files: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `scripts/prepare-release-notes.sh`, `scripts/verify-release-config.sh`
- Risk: Script regressions or release-environment drift can land on `main` without detection because the default CI job only runs `go test ./...`.
- Priority: High

**The default external-editor launch path is effectively untested:**
- What's not tested: `defaultEditFile` with real `$VISUAL` / `$EDITOR` command strings and a real TTY-backed editor invocation.
- Files: `internal/cli/create.go`, `internal/cli/app_integration_test.go`
- Risk: Interactive users hit editor-launch failures that do not show up in automated tests because the integration test replaces `app.EditFile` with a stub.
- Priority: Medium

---

*Concerns audit: 2026-04-06*
