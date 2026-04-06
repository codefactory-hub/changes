# Architecture

**Analysis Date:** 2026-04-06

## Pattern Overview

**Overall:** Layered Go CLI with a thin executable entry point, a command-parsing CLI layer, an application service layer, and small domain/storage packages under `internal/`.

**Key Characteristics:**
- Keep `cmd/changes/main.go` minimal. It wires stdout/stderr, passes `os.Args`, and exits on error.
- Put command parsing and terminal UX in `internal/cli/`. Keep business rules out of flag handling.
- Put use-case orchestration in `internal/app/`. That layer loads config and state, selects domain operations, and returns plain result structs.
- Keep durable release data in repo-local XDG-style paths resolved through `internal/config/config.go`.
- Treat fragments and release records as append-only source data. Rendered outputs are views, not source of truth.
- Model release history through parent-linked base release records in `internal/releases/releases.go`, then assemble renderable bundles in `internal/releases/bundle.go`.

## Layers

**Executable Layer:**
- Purpose: Start the CLI process and bridge process-level state into the app.
- Location: `cmd/changes/main.go`
- Contains: `main()`, build metadata variables (`version`, `commit`, `date`)
- Depends on: `internal/cli`
- Used by: `go run ./cmd/changes`, GoReleaser `builds.main` in `.goreleaser.yaml`

**CLI Layer:**
- Purpose: Parse commands and flags, own help text, interactive prompts, editor launching, output formatting, and repo-root detection.
- Location: `internal/cli/app.go`, `internal/cli/create.go`, `internal/cli/stringslice.go`
- Contains: `App.Run`, `runInit`, `runCreate`, `runStatus`, `runRelease`, `runRender`
- Depends on: `internal/app`, `internal/config`, `internal/render`, `internal/reporoot`, `internal/semverpolicy`, `internal/versioning`
- Used by: `cmd/changes/main.go`

**Application Service Layer:**
- Purpose: Implement command use cases as pure orchestration over config, fragment state, release state, and rendering.
- Location: `internal/app/app.go`, `internal/app/init.go`
- Contains: `Initialize`, `Status`, `PlanRelease`, `CommitRelease`, `Render`
- Depends on: `internal/config`, `internal/fragments`, `internal/releases`, `internal/render`, `internal/semverpolicy`, `internal/versioning`
- Used by: `internal/cli/app.go`

**Configuration and Repository Layout Layer:**
- Purpose: Resolve repo-relative paths, defaults, render-profile config, and validation.
- Location: `internal/config/config.go`, `.config/changes/config.toml`
- Contains: `Config`, `RenderProfile`, path helpers such as `FragmentsDir`, `ReleasesDir`, `TemplatesDir`, `ChangelogPath`
- Depends on: `BurntSushi/toml`, stdlib filesystem/path packages
- Used by: Nearly every application/domain package

**Fragment Storage Layer:**
- Purpose: Create, parse, validate, load, and list fragment files.
- Location: `internal/fragments/fragments.go`
- Contains: `Fragment`, `Metadata`, `Create`, `Load`, `List`, `Parse`, `Format`
- Depends on: `internal/config`, `BurntSushi/toml`
- Used by: `internal/app`, `internal/releases`, tests

**Release Model Layer:**
- Purpose: Persist release records, validate lineage rules, select unreleased fragment sets, and compute parent relationships.
- Location: `internal/releases/releases.go`, `internal/releases/bundle.go`
- Contains: `ReleaseRecord`, `ReleaseBundle`, `ValidateSet`, `Lineage`, `UnreleasedFinalFragments`, `UnreleasedPrereleaseFragments`, `AssembleRelease`, `AssembleReleaseLineage`
- Depends on: `internal/config`, `internal/fragments`, `internal/versioning`, `BurntSushi/toml`
- Used by: `internal/app`, `internal/changelog`, `internal/render`

**Version and Policy Layer:**
- Purpose: Parse/compare versions and convert fragment metadata into a suggested bump.
- Location: `internal/versioning/versioning.go`, `internal/semverpolicy/policy.go`
- Contains: version parsing/comparison helpers plus recommendation logic keyed by `versioning.public_api`
- Depends on: `Masterminds/semver/v3`, `internal/fragments`
- Used by: `internal/app`, `internal/releases`, CLI explanation output

**Rendering Layer:**
- Purpose: Resolve render profiles, load templates, assemble documents, enforce profile limits, and generate user-facing output.
- Location: `internal/render/render.go`, `internal/render/profiles.go`, `internal/render/templates_builtin.go`
- Contains: `Renderer`, `TemplatePack`, built-in profiles, built-in template bodies
- Depends on: `internal/config`, `internal/releases`, Go `text/template`
- Used by: `internal/app.Render`, `internal/changelog`, `internal/templates`

**Changelog Materialization Layer:**
- Purpose: Rebuild and write the repository changelog from the latest stable lineage.
- Location: `internal/changelog/changelog.go`
- Contains: `Rebuild`, `Write`
- Depends on: `internal/config`, `internal/fragments`, `internal/releases`, `internal/render`
- Used by: Tests now; the public CLI render path can generate equivalent output via `changes render --latest --profile repository_markdown`

**Template Bootstrap Layer:**
- Purpose: Seed repo-local template override files from built-in templates.
- Location: `internal/templates/defaults.go`
- Contains: `EnsureDefaultFiles`
- Depends on: `internal/config`, `internal/render`
- Used by: Internal/template tests now; available for future bootstrap flows

**Environment Detection Layer:**
- Purpose: Detect the enclosing Git repository root before any repo-local paths are used.
- Location: `internal/reporoot/reporoot.go`
- Contains: `Detect`
- Depends on: stdlib path/filesystem
- Used by: `internal/cli/app.go`

## Data Flow

**Create Fragment Flow:**

1. `cmd/changes/main.go` calls `internal/cli.NewApp(...).Run(...)`.
2. `internal/cli/app.go` dispatches `create` to `runCreate`.
3. `internal/cli/create.go` parses flags, optionally prompts or opens an editor, then loads config via `internal/config/config.go`.
4. `internal/fragments/fragments.go` normalizes metadata, allocates a unique fragment ID, and writes a Markdown file under `.local/share/changes/fragments/`.
5. The CLI prints the created fragment path.

**Status and Release Planning Flow:**

1. `internal/cli/app.go` resolves the repo root with `internal/reporoot/reporoot.go`.
2. `internal/app/app.go` loads config and current state via `fragments.List` and `releases.List`.
3. `internal/releases/releases.go` computes unreleased fragment sets from lineage reachability.
4. `internal/semverpolicy/policy.go` maps pending fragment levers to a recommended bump.
5. `internal/versioning/versioning.go` derives the next stable or prerelease version.
6. The CLI formats a human-readable plan or explanation.

**Commit Release Flow:**

1. `internal/app.PlanRelease` selects fragments and parent version for the chosen target.
2. `internal/app.CommitRelease` reloads current state and rejects stale plans by comparing fragment IDs and parent lineage.
3. `internal/releases/releases.go` validates the full release set and writes a new base record under `.local/share/changes/releases/`.
4. The CLI prints the created record path.

**Render Flow:**

1. `internal/cli/app.go` parses `changes render` selectors: `--version`, `--record`, or `--latest`.
2. `internal/app.Render` loads config, fragments, and release records.
3. `internal/releases/releases.go` selects the target base record and `internal/releases/bundle.go` assembles one `ReleaseBundle` or a lineage of bundles.
4. `internal/render/profiles.go` resolves the render profile from `.config/changes/config.toml`.
5. `internal/render/render.go` loads repo-local template overrides from `.local/share/changes/templates/` when present, otherwise falls back to `internal/render/templates_builtin.go`.
6. The renderer emits the final content, optionally trimming whole release blocks when `max_chars` applies to chain renders.

**Release and Distribution Pipeline:**

1. `scripts/prepare-release-notes.sh` calls `go run ./cmd/changes render --latest --profile repository_markdown --output .dist/release-notes.md`, with a placeholder fallback when no stable release exists.
2. `.github/workflows/release.yml` runs that script on pushed `v*` tags.
3. `.goreleaser.yaml` builds the `cmd/changes` binary, injects build metadata into `main.version`, `main.commit`, and `main.date`, publishes a GitHub Release, and updates a Homebrew cask tap.
4. `.github/workflows/release-dry-run.yml` rehearses the same path without publishing, using `scripts/verify-release-config.sh` and `scripts/build-release-snapshot.sh`.

**State Management:**
- Persistent state is file-backed and repo-local, not database-backed.
- Durable source data lives under `.local/share/changes/` and is loaded fresh per command.
- No long-lived in-memory state or daemon exists. Each CLI invocation recomputes state from files.

## Key Abstractions

**Fragment:**
- Purpose: Durable, human-editable source record for one release-relevant change.
- Examples: `internal/fragments/fragments.go`, `.local/share/changes/fragments/20260406-134200--release-automation--gentle-island-keeps.md`
- Pattern: Markdown body with TOML front matter plus normalized metadata fields.

**ReleaseRecord:**
- Purpose: Canonical per-release selection record that freezes which fragment IDs were added by that release step.
- Examples: `internal/releases/releases.go`, `.local/share/changes/releases/`
- Pattern: TOML file keyed by `<product>-<version>.toml`, with strict validation for base vs companion records.

**ReleaseBundle:**
- Purpose: Render-time assembled release view: base record, companion records, lineage context, and sectioned fragment entries.
- Examples: `internal/releases/bundle.go`
- Pattern: Derived aggregate object; never written as source data.

**RenderProfile / TemplatePack:**
- Purpose: Declarative rendering contract that maps a profile name to mode, templates, metadata, and char limits.
- Examples: `.config/changes/config.toml`, `internal/render/profiles.go`, `internal/render/render.go`
- Pattern: Config-backed profile resolved to a concrete template pack at runtime.

**Repository-Local XDG Layout:**
- Purpose: Keep source-of-truth data reviewable in the repo while separating durable vs transient artifacts.
- Examples: `internal/config/config.go`, `docs/decisions/ADR-0001-repo-local-xdg-layout.md`
- Pattern: Config under `.config/`, durable data under `.local/share/`, transient state under `.local/state/`.

## Entry Points

**CLI Binary:**
- Location: `cmd/changes/main.go`
- Triggers: `go run ./cmd/changes`, compiled `changes` binary, GoReleaser builds
- Responsibilities: Instantiate `internal/cli.App`, pass process args, exit on failure

**Command Router:**
- Location: `internal/cli/app.go`
- Triggers: All CLI invocations
- Responsibilities: Dispatch commands, parse flags, prompt in TTY mode, format output, detect repo root

**Release Notes Preparation Script:**
- Location: `scripts/prepare-release-notes.sh`
- Triggers: Local release rehearsal and GitHub release workflows
- Responsibilities: Produce `.dist/release-notes.md` from the latest stable render, or generate a placeholder

**GitHub Release Workflow:**
- Location: `.github/workflows/release.yml`
- Triggers: Git tag pushes matching `v*`, manual workflow dispatch
- Responsibilities: Checkout, set up Go, prepare release notes, invoke GoReleaser publish

**Dry-Run Release Workflow:**
- Location: `.github/workflows/release-dry-run.yml`
- Triggers: Manual workflow dispatch
- Responsibilities: Validate release config, run snapshot builds, upload rehearsal artifacts

## Error Handling

**Strategy:** Return errors upward with contextual wrapping; print exactly once at the CLI boundary.

**Patterns:**
- Domain and service packages return `error` with `fmt.Errorf(...: %w)`, as in `internal/app/app.go`, `internal/config/config.go`, and `internal/releases/releases.go`.
- `internal/cli/app.go` centralizes final stderr formatting through `App.fail`.
- Validation is fail-fast and file-backed commands stop before partial writes unless an init transaction is explicitly managing rollback in `internal/app/init.go`.
- `internal/app.CommitRelease` defends against stale release plans by reloading state and comparing fragment selections before writing.

## Cross-Cutting Concerns

**Logging:** No structured logging layer is present. Commands print user-facing output to stdout/stderr in `internal/cli/app.go`, and shell scripts emit simple stderr messages.

**Validation:** Validation is package-local and explicit. `internal/fragments/fragments.go`, `internal/config/config.go`, and `internal/releases/releases.go` each validate their own models before writing or using data.

**Authentication:** No application-level authentication exists. The only authenticated path is release automation via environment-backed GitHub/Homebrew credentials consumed by `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, and `.goreleaser.yaml`.

**Filesystem Boundaries:** Treat repo-root detection and config path helpers as mandatory. New code should not hardcode `.local/...` paths outside `internal/config/config.go`.

**Rendering as a View:** Keep renderers read-only with respect to fragment/release source data. `internal/render/` and `internal/changelog/` should consume `ReleaseBundle` data, not mutate records or fragments.

---

*Architecture analysis: 2026-04-06*
