# Releasing `changes`

## Overview

The repository ships a Go binary with GoReleaser and publishes a Homebrew cask into a private/internal tap repository.

The current configuration is intentionally placeholder-driven where organization-specific information would otherwise be required.

## Required environment variables

- `GITHUB_TOKEN`
  Used by GoReleaser to create GitHub releases in this repository.
- `HOMEBREW_TAP_GITHUB_TOKEN`
  Token with push access to the private tap repository.
- `HOMEBREW_TAP_OWNER`
  Tap repository owner or organization.
- `HOMEBREW_TAP_REPO`
  Tap repository name.
- `HOMEBREW_TAP_NAME`
  Homebrew tap name, for example `internal-tools`.
- `RELEASE_DOWNLOAD_URL_BASE`
  Base URL for published release artifacts.

## Tagging flow

1. Ensure `go test ./...` is green.
2. Create and push a semantic version tag such as `v0.1.0`.
3. Let `.github/workflows/release.yml` run GoReleaser.

## Intended self-hosted release-notes path

The release workflow includes a guarded step that runs:

```bash
go run ./cmd/changes changelog rebuild
```

That step is currently conditional on `.config/changes/config.toml` existing in this repository. Once this repository begins using `changes` for its own fragments and manifests, that hook becomes the path to generating the project changelog from durable release metadata.
