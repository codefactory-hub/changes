# External Integrations

**Analysis Date:** 2026-04-06

## APIs & External Services

**Release Publishing:**
- GitHub Releases - release artifacts are published by GoReleaser from `.goreleaser.yaml`.
  - SDK/Client: `goreleaser/goreleaser-action@v7` in `.github/workflows/release.yml`
  - Auth: `GITHUB_TOKEN` in `.github/workflows/release.yml` and `docs/releasing/RELEASING.md`
- GitHub REST API - the generated Homebrew cask resolves release asset URLs via `https://api.github.com/repos/{owner}/{repo}/releases/tags/{tag}` in the Ruby `custom_block` inside `.goreleaser.yaml`.
  - SDK/Client: Ruby `net/http` code embedded in `.goreleaser.yaml`
  - Auth: `HOMEBREW_GITHUB_API_TOKEN` or Homebrew GitHub credentials, as referenced in `.goreleaser.yaml` and `docs/releasing/RELEASING.md`

**Package Distribution:**
- Private Homebrew tap repository - GoReleaser updates a separate cask repository through the `homebrew_casks` block in `.goreleaser.yaml`.
  - SDK/Client: GoReleaser `homebrew_casks` support configured in `.goreleaser.yaml`
  - Auth: `CHANGES_HOMEBREW_TAP_TOKEN` in `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, and `docs/releasing/RELEASING.md`

**CI/CD Services:**
- GitHub Actions - CI, dry-run, and tagged-release automation are defined in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, and `.github/workflows/release.yml`.
  - SDK/Client: `actions/checkout@v5`, `actions/setup-go@v6`, `goreleaser/goreleaser-action@v7`
  - Auth: GitHub-hosted workflow permissions plus `GITHUB_TOKEN` and repo vars/secrets in the workflow files

**Developer Tooling:**
- Local editor process - `changes create --edit` launches the configured editor from `internal/cli/create.go`.
  - SDK/Client: Go `os/exec` in `internal/cli/create.go`
  - Auth: none
- Git repository layout - commands require a repository with a `.git` marker, detected in `internal/reporoot/reporoot.go`.
  - SDK/Client: filesystem checks in `internal/reporoot/reporoot.go`
  - Auth: none

## Data Storage

**Databases:**
- Local filesystem only. Repo configuration is stored in `.config/changes/config.toml`; release fragments are stored under `.local/share/changes/fragments`; release records are stored under `.local/share/changes/releases`; prompts are stored under `.local/share/changes/prompts`; path resolution is implemented in `internal/config/config.go`.
  - Connection: Not applicable
  - Client: Go standard library file IO plus `github.com/BurntSushi/toml` in `internal/config/config.go`, `internal/fragments/fragments.go`, and `internal/releases/releases.go`

**File Storage:**
- Local filesystem only. Initialization creates repo-local directories and starter files in `internal/app/init.go`; rendering can also read repo-local template overrides from `.local/share/changes/templates` through `internal/render/render.go`.

**Caching:**
- None for application runtime. Build-time caching is used only in release rehearsal via repo-local Go cache paths in `scripts/build-release-snapshot.sh`, and GitHub-hosted Go cache support is enabled by `actions/setup-go@v6` in `.github/workflows/release-dry-run.yml` and `.github/workflows/release.yml`.

## Authentication & Identity

**Auth Provider:**
- None for the CLI application itself. No OAuth, session, or identity provider integration is detected under `cmd/` or `internal/`.
  - Implementation: unauthenticated local CLI; authenticated operations are limited to release infrastructure tokens in `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, and `.goreleaser.yaml`

## Monitoring & Observability

**Error Tracking:**
- None detected. No Sentry, Honeycomb, Datadog, OpenTelemetry, or similar client imports are present under `cmd/`, `internal/`, or `scripts/`.

**Logs:**
- Standard stdout/stderr CLI output. User-facing command output is written by `internal/cli/app.go`; script diagnostics are printed with shell `echo`/`printf` in `scripts/prepare-release-notes.sh` and `scripts/verify-release-config.sh`.

## CI/CD & Deployment

**Hosting:**
- Source hosting and release hosting are GitHub-based, inferred from `.github/workflows/*.yml`, `.goreleaser.yaml`, and `docs/releasing/RELEASING.md`.
- End-user package distribution includes a private Homebrew tap configured in `.goreleaser.yaml`.

**CI Pipeline:**
- `ci` workflow runs `go test ./...` on pushes to `main` and pull requests in `.github/workflows/ci.yml`.
- `release-dry-run` workflow validates release notes, verifies GoReleaser config, builds a snapshot, and uploads artifacts in `.github/workflows/release-dry-run.yml`.
- `release` workflow runs on `v*` tags or manual dispatch and publishes release artifacts in `.github/workflows/release.yml`.

## Environment Configuration

**Required env vars:**
- `CHANGES_GITHUB_OWNER` - release repository owner; used by `.goreleaser.yaml`, `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, `scripts/verify-release-config.sh`, and `scripts/build-release-snapshot.sh`
- `CHANGES_GITHUB_REPO` - release repository name; used by the same files as `CHANGES_GITHUB_OWNER`
- `CHANGES_PROJECT_HOMEPAGE` - Homebrew cask homepage; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_HOMEBREW_TAP_OWNER` - tap owner; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_HOMEBREW_TAP_REPO` - tap repository; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_HOMEBREW_TAP_BRANCH` - tap branch; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_RELEASE_BOT_NAME` - commit author name for tap updates; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_RELEASE_BOT_EMAIL` - commit author email for tap updates; used by `.goreleaser.yaml`, workflow files, and release scripts
- `CHANGES_HOMEBREW_TAP_TOKEN` - secret token for writing to the tap repo; documented in `docs/releasing/RELEASING.md` and used in workflow files plus `.goreleaser.yaml`
- `GITHUB_TOKEN` - GitHub Actions release token; used in `.github/workflows/release.yml`
- `HOMEBREW_GITHUB_API_TOKEN` - optional local token for private-tap installs; documented in `docs/releasing/RELEASING.md` and referenced in `.goreleaser.yaml`
- `VISUAL` or `EDITOR` - optional local editor configuration for `changes create --edit`; read by `internal/cli/create.go`

**Secrets location:**
- GitHub repository variables and secrets, as documented in `docs/releasing/RELEASING.md` and referenced in `.github/workflows/release.yml` and `.github/workflows/release-dry-run.yml`.
- Local shell environment for `VISUAL`/`EDITOR` and optional `HOMEBREW_GITHUB_API_TOKEN`.

## Webhooks & Callbacks

**Incoming:**
- None detected as application endpoints. The repo receives automation triggers only through GitHub Actions events in `.github/workflows/ci.yml`, `.github/workflows/release-dry-run.yml`, and `.github/workflows/release.yml`.

**Outgoing:**
- GitHub Release publication via GoReleaser from `.goreleaser.yaml` and `.github/workflows/release.yml`
- GitHub REST API calls for release asset lookup from the generated Homebrew cask logic in `.goreleaser.yaml`
- Homebrew tap repository updates via the `homebrew_casks` repository block in `.goreleaser.yaml`

---

*Integration audit: 2026-04-06*
