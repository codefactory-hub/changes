# Releasing `changes`

This repository is set up to publish tagged releases through GoReleaser and then push a Homebrew cask into a separate internal tap repository.

## Expected release flow

1. Push a `v*` tag to this repository.
2. The GitHub Actions workflow generates release notes.
3. GoReleaser builds the `changes` binary for Linux, macOS, and Windows.
4. GoReleaser publishes a GitHub Release and updates the internal Homebrew cask tap.

## Required repository variables

Set these as GitHub repository variables unless your hosting provider uses a different secret model:

- `CHANGES_GITHUB_OWNER`
- `CHANGES_GITHUB_REPO`
- `CHANGES_PROJECT_HOMEPAGE`
- `CHANGES_HOMEBREW_TAP_OWNER`
- `CHANGES_HOMEBREW_TAP_REPO`
- `CHANGES_HOMEBREW_TAP_BRANCH`
- `CHANGES_RELEASE_BOT_NAME`
- `CHANGES_RELEASE_BOT_EMAIL`

## Required secrets

- `GITHUB_TOKEN` is provided by GitHub Actions for the release repository.
- `CHANGES_HOMEBREW_TAP_TOKEN` must have write access to the tap repository.
- Homebrew installs from the private tap may also require `HOMEBREW_GITHUB_API_TOKEN` on the user machine, depending on how the private cask is accessed.

## Notes generation

The workflow currently tries to run:

```bash
go run ./cmd/changes changelog rebuild --output .dist/release-notes.md
```

If the CLI is not present yet, the workflow writes a placeholder file so release automation stays coherent during bootstrap.

## Tap model

The cask config is intentionally template-driven and uses placeholders for:

- the release repository owner/name
- the tap repository owner/name/branch
- the tap repository token
- the release homepage URL

This keeps the release process portable without hardcoding organization-specific values into `.goreleaser.yaml`.
