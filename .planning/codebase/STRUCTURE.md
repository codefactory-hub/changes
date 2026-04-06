# Codebase Structure

**Analysis Date:** 2026-04-06

## Directory Layout

```text
changes/
├── cmd/                     # Binary entrypoints
├── internal/                # Private application and domain packages
├── scripts/                 # Release and verification shell scripts
├── .github/workflows/       # CI and release automation
├── .config/changes/         # Repo-local configuration
├── .local/share/changes/    # Durable repo-local source data
├── .local/state/changes/    # Transient state and caches
├── docs/                    # ADRs, plans, release docs, research
├── testdata/                # Reserved test fixture root; currently empty
├── .planning/codebase/      # Generated codebase mapping docs
├── .goreleaser.yaml         # Release build and publish definition
├── go.mod                   # Module and dependency definition
└── README.md                # Product and workflow documentation
```

## Directory Purposes

**`cmd/`:**
- Purpose: Hold executable entrypoints only.
- Contains: `cmd/changes/main.go`
- Key files: `cmd/changes/main.go`
- Use it for: New binaries or alternate entrypoints. Keep orchestration elsewhere.

**`internal/app/`:**
- Purpose: Use-case orchestration layer.
- Contains: command-facing service functions and init transaction logic
- Key files: `internal/app/app.go`, `internal/app/init.go`
- Use it for: New workflows that combine config, storage, policy, and rendering.

**`internal/cli/`:**
- Purpose: CLI command routing, flag parsing, help text, prompts, and terminal I/O.
- Contains: command handlers and CLI helper types
- Key files: `internal/cli/app.go`, `internal/cli/create.go`, `internal/cli/stringslice.go`
- Use it for: New command surfaces, flags, prompt text, and stdout/stderr formatting.

**`internal/config/`:**
- Purpose: Repo-local config model and path resolution.
- Contains: config structs, path helpers, TOML load/validate logic
- Key files: `internal/config/config.go`
- Use it for: Any new configurable path or render-profile setting.

**`internal/fragments/`:**
- Purpose: Fragment file lifecycle.
- Contains: create/load/list/parse/format logic and fragment validation
- Key files: `internal/fragments/fragments.go`
- Use it for: New fragment metadata fields or fragment file rules.

**`internal/releases/`:**
- Purpose: Release record persistence, lineage rules, unreleased selection, and render bundle assembly.
- Contains: release record model plus bundle-building helpers
- Key files: `internal/releases/releases.go`, `internal/releases/bundle.go`
- Use it for: New release metadata, lineage rules, or sectioning logic.

**`internal/render/`:**
- Purpose: Render profile resolution and template execution.
- Contains: renderer, built-in profiles, built-in templates
- Key files: `internal/render/render.go`, `internal/render/profiles.go`, `internal/render/templates_builtin.go`
- Use it for: New output formats or profile/template behavior.

**`internal/changelog/`:**
- Purpose: Rebuild and persist the repository changelog.
- Contains: changelog materialization helpers
- Key files: `internal/changelog/changelog.go`
- Use it for: Changelog-specific write flows.

**`internal/templates/`:**
- Purpose: Seed repo-local override templates.
- Contains: template bootstrap helper
- Key files: `internal/templates/defaults.go`
- Use it for: Initialization/bootstrap work that needs default template files on disk.

**`internal/versioning/`:**
- Purpose: SemVer parsing and version arithmetic.
- Contains: version model, compare helpers, bump helpers
- Key files: `internal/versioning/versioning.go`
- Use it for: Any version math. Do not duplicate SemVer logic elsewhere.

**`internal/semverpolicy/`:**
- Purpose: Convert fragment semantics into a recommended bump.
- Contains: release recommendation policy
- Key files: `internal/semverpolicy/policy.go`
- Use it for: Policy changes tied to `versioning.public_api`.

**`internal/reporoot/`:**
- Purpose: Git repo-root detection.
- Contains: `Detect`
- Key files: `internal/reporoot/reporoot.go`
- Use it for: Any command that needs to operate from arbitrary subdirectories.

**`internal/collection/`:**
- Purpose: Reserved internal package area.
- Contains: No Go files currently
- Key files: Not applicable
- Use it for: Nothing by default. Avoid placing new code here unless a new collection feature is explicitly introduced.

**`scripts/`:**
- Purpose: Shell entrypoints for release-note generation and release verification.
- Contains: `scripts/prepare-release-notes.sh`, `scripts/verify-release-config.sh`, `scripts/build-release-snapshot.sh`
- Key files: those three scripts
- Use it for: Release automation helpers that are simpler in shell than in Go.

**`.github/workflows/`:**
- Purpose: CI and release automation.
- Contains: `ci.yml`, `release.yml`, `release-dry-run.yml`
- Key files: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`
- Use it for: GitHub-triggered orchestration only; keep reusable logic in scripts or Go code.

**`.config/changes/`:**
- Purpose: Repo-local committed configuration.
- Contains: `config.toml`
- Key files: `.config/changes/config.toml`
- Use it for: Defaults, render profiles, product name, changelog path, and data/state/template root paths.

**`.local/share/changes/`:**
- Purpose: Durable committed release data.
- Contains: `fragments/`, `releases/`, and optionally `prompts/` and `templates/`
- Key files: `.local/share/changes/fragments/20260406-133900--core-release-model--brave-harbor-guides.md`
- Key files: `.local/share/changes/releases/.gitkeep`
- Use it for: Canonical source records, not transient caches.

**`.local/state/changes/`:**
- Purpose: Transient repo-local state.
- Contains: cache and state files such as `.local/state/changes/catalog-check.json`
- Key files: `.local/state/changes/catalog-check.json`
- Use it for: Ephemeral state, caches, and generated diagnostics that are not core release history.

**`docs/`:**
- Purpose: Product decisions and process docs.
- Contains: ADRs in `docs/decisions/`, plans in `docs/plans/`, release docs in `docs/releasing/`, research in `docs/research/`
- Key files: `docs/decisions/ADR-0001-repo-local-xdg-layout.md`, `docs/releasing/RELEASING.md`
- Use it for: Architectural rationale and release process documentation.

## Key File Locations

**Entry Points:**
- `cmd/changes/main.go`: Public binary entrypoint.
- `internal/cli/app.go`: Command router and CLI help surface.
- `scripts/prepare-release-notes.sh`: Release-notes generation entrypoint for automation.
- `.github/workflows/release.yml`: Publish workflow entrypoint.

**Configuration:**
- `go.mod`: Go module path and dependencies.
- `.config/changes/config.toml`: Repo-local product, path, and render-profile configuration.
- `.goreleaser.yaml`: Release build and distribution definition.

**Core Logic:**
- `internal/app/app.go`: Service orchestration for status, release, and render flows.
- `internal/app/init.go`: Repository initialization and bootstrap prompt generation.
- `internal/fragments/fragments.go`: Fragment file lifecycle.
- `internal/releases/releases.go`: Release record persistence and lineage.
- `internal/releases/bundle.go`: Render-time release assembly.
- `internal/render/render.go`: Template execution and output shaping.
- `internal/versioning/versioning.go`: Version parsing and bump math.
- `internal/semverpolicy/policy.go`: Recommendation policy.

**Testing:**
- `internal/cli/app_integration_test.go`: Integration tests for the CLI surface.
- `internal/app/app_test.go`: Service-layer tests.
- `internal/render/render_test.go`: Renderer behavior tests.
- `internal/releases/releases_test.go`: Release lineage and selection tests.

## Naming Conventions

**Files:**
- Use lowercase package directories and lowercase Go filenames, usually matching the package concern: `internal/releases/releases.go`, `internal/render/render.go`.
- Use `_test.go` alongside the package under test: `internal/versioning/versioning_test.go`.
- Use command subdirectories under `cmd/` named for the binary: `cmd/changes/`.

**Directories:**
- Keep package boundaries flat and purpose-driven under `internal/`.
- Use singular concern names for packages: `app`, `config`, `render`, `releases`, `fragments`.
- Keep repo-local data under XDG-style roots instead of ad hoc top-level folders: `.config/changes/`, `.local/share/changes/`, `.local/state/changes/`.

## Where to Add New Code

**New Feature:**
- Primary code: `internal/app/` for the use case, plus the relevant domain package under `internal/`
- CLI surface: `internal/cli/`
- Tests: co-locate in the same package with `_test.go`

**New Command:**
- Implementation: add a `run<Command>` method in `internal/cli/app.go` or a focused helper file in `internal/cli/`
- Orchestration: add a service entrypoint in `internal/app/app.go` if the command does more than pure I/O formatting
- Avoid: Putting command-specific business rules into `cmd/changes/main.go`

**New Fragment Metadata or Parsing Rule:**
- Implementation: `internal/fragments/fragments.go`
- Downstream consumers to update: `internal/semverpolicy/policy.go`, `internal/releases/bundle.go`, templates in `internal/render/templates_builtin.go` or `.local/share/changes/templates/`

**New Release Metadata or Lineage Rule:**
- Implementation: `internal/releases/releases.go`
- Render-side follow-up: `internal/releases/bundle.go` and `internal/render/`

**New Render Profile or Output Format:**
- Built-in profile registration: `internal/render/profiles.go`
- Built-in template body: `internal/render/templates_builtin.go`
- Config override wiring: `.config/changes/config.toml`
- Do not: Hardcode new rendering behavior in CLI handlers

**Utilities:**
- Shared helpers: place them in the narrowest existing package that owns the concern
- New package creation: use a new `internal/<concern>/` package only when the concern does not cleanly belong to an existing package

## Special Directories

**`.planning/codebase/`:**
- Purpose: Generated mapping/reference docs for planning agents
- Generated: Yes
- Committed: Intended to be committed

**`.local/share/changes/fragments/`:**
- Purpose: Durable fragment artifacts
- Generated: Yes, by `changes create` and bootstrap flows
- Committed: Yes

**`.local/share/changes/releases/`:**
- Purpose: Canonical release records
- Generated: Yes, by `changes release` and bootstrap flows
- Committed: Yes

**`.local/share/changes/prompts/`:**
- Purpose: Repo-specific bootstrap/import prompts
- Generated: Yes, by `changes init` when adopting an existing product history
- Committed: Yes

**`.local/share/changes/templates/`:**
- Purpose: Repo-local template overrides
- Generated: Optional, via `internal/templates/defaults.go`
- Committed: Yes when present

**`.local/state/changes/`:**
- Purpose: Transient state, caches, and diagnostics
- Generated: Yes
- Committed: Not a source-of-truth area; treat as ephemeral

**`docs/decisions/`:**
- Purpose: ADR record for architecture decisions
- Generated: No
- Committed: Yes

**`testdata/`:**
- Purpose: Fixture root for tests
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-04-06*
