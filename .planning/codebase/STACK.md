# Technology Stack

**Analysis Date:** 2026-04-06

## Languages

**Primary:**
- Go 1.26.0 - application code and tests live in `go.mod`, `cmd/changes/main.go`, and the packages under `internal/`.

**Secondary:**
- Bash - release automation and local verification scripts live in `scripts/prepare-release-notes.sh`, `scripts/verify-release-config.sh`, and `scripts/build-release-snapshot.sh`.
- YAML - CI/CD and release automation are declared in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, `.github/workflows/release.yml`, and `.goreleaser.yaml`.
- TOML - repo configuration and persisted release data are defined in `.config/changes/config.toml`, parsed by `internal/config/config.go`, and written/read by `internal/fragments/fragments.go` and `internal/releases/releases.go`.
- Markdown/templates - user-facing docs and built-in render templates live in `README.md`, `docs/releasing/RELEASING.md`, and `internal/render/templates_builtin.go`.

## Runtime

**Environment:**
- Native Go CLI runtime targeting local execution. The only executable entry point is `cmd/changes/main.go`.
- Commands assume execution inside a Git repository; repo-root detection is implemented in `internal/reporoot/reporoot.go`.

**Package Manager:**
- Go modules - dependency management is declared in `go.mod`.
- Lockfile: missing. No `go.sum` is present in the repository root.

## Frameworks

**Core:**
- Go standard library CLI stack - command parsing uses `flag` in `internal/cli/app.go` and `internal/cli/create.go`; there is no third-party CLI framework such as Cobra detected.
- Internal package architecture - business logic is split across `internal/app/`, `internal/config/`, `internal/fragments/`, `internal/releases/`, `internal/render/`, `internal/semverpolicy/`, and `internal/versioning/`.

**Testing:**
- Go `testing` package - unit and integration coverage is implemented in files such as `internal/app/app_test.go`, `internal/cli/app_integration_test.go`, `internal/render/render_test.go`, and `internal/versioning/versioning_test.go`.

**Build/Dev:**
- Go toolchain - development and test entrypoint is `go test ./...`, documented in `README.md` and used in `.github/workflows/ci.yml`.
- GoReleaser v2 configuration - binary packaging and release publishing are defined in `.goreleaser.yaml`; local verification in `scripts/verify-release-config.sh` requires GoReleaser v2.10+.
- GitHub Actions - CI and release pipelines are defined in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, and `.github/workflows/release.yml`.

## Key Dependencies

**Critical:**
- `github.com/BurntSushi/toml` v1.6.0 - canonical TOML parser/encoder for repo config, fragment front matter, and release records in `internal/config/config.go`, `internal/fragments/fragments.go`, `internal/releases/releases.go`, and `internal/app/init.go`.
- `github.com/Masterminds/semver/v3` v3.4.0 - SemVer parsing and ordering authority in `internal/versioning/versioning.go`, consumed by release planning in `internal/app/app.go` and policy logic in `internal/semverpolicy/policy.go`.

**Infrastructure:**
- `actions/checkout@v5` - source checkout in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, and `.github/workflows/release.yml`.
- `actions/setup-go@v6` - Go toolchain setup and caching in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, and `.github/workflows/release.yml`.
- `goreleaser/goreleaser-action@v7` - release and dry-run automation in `.github/workflows/release-dry-run.yml` and `.github/workflows/release.yml`.

## Configuration

**Environment:**
- Repo-local configuration is loaded from `.config/changes/config.toml` by `internal/config/config.go`; this file defines project metadata, storage paths, render profiles, and versioning policy.
- Repo-local data/state directories default to `.local/share/changes` and `.local/state/changes`, as defined in `internal/config/config.go` and initialized by `internal/app/init.go`.
- Interactive editing depends on `$VISUAL` or `$EDITOR`, read in `internal/cli/create.go`.
- Release automation depends on GitHub/Homebrew environment variables documented in `docs/releasing/RELEASING.md` and consumed in `.github/workflows/release-dry-run.yml`, `.github/workflows/release.yml`, `scripts/verify-release-config.sh`, and `scripts/build-release-snapshot.sh`.

**Build:**
- Module and toolchain version: `go.mod`.
- Release packaging: `.goreleaser.yaml`.
- CI/release orchestration: `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, `.github/workflows/release.yml`.
- Local release helpers: `scripts/prepare-release-notes.sh`, `scripts/verify-release-config.sh`, `scripts/build-release-snapshot.sh`.

## Platform Requirements

**Development:**
- Go 1.26.0-compatible toolchain from `go.mod`.
- Git working tree required for command execution because repo detection depends on a `.git` marker in `internal/reporoot/reporoot.go`.
- Optional local editor integration requires `$VISUAL` or `$EDITOR` for `changes create --edit`, implemented in `internal/cli/create.go`.
- Local release rehearsal requires GoReleaser v2.10+ per `scripts/verify-release-config.sh` and `docs/releasing/RELEASING.md`.

**Production:**
- Distribution target is a compiled cross-platform CLI. `.goreleaser.yaml` builds `changes` from `./cmd/changes` for `darwin`, `linux`, and `windows` on `amd64` and `arm64`.
- Published artifacts are GitHub release assets plus a private Homebrew cask path, as configured in `.goreleaser.yaml` and described in `docs/releasing/RELEASING.md`.

---

*Stack analysis: 2026-04-06*
