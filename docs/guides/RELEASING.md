# Releasing `changes`

This repository is set up to publish tagged releases through GoReleaser and then push a Homebrew cask into a separate internal tap repository.

This guide follows [ADR-0006: Use Homebrew cask distribution for internal releases](../03-decisions/ADR-0006-homebrew-cask-distribution.md), [ADR-0008: Separate human release auth from automation and agent auth](../03-decisions/ADR-0008-separate-human-release-auth-from-automation-and-agent-auth.md), and [ADR-0009: Adopt provider-neutral release secret ingestion](../03-decisions/ADR-0009-adopt-provider-neutral-release-secret-ingestion.md).

## Expected release flow

1. Push a `v*` tag to this repository.
2. The GitHub Actions workflow reads the committed GitHub release markdown for that version.
3. GoReleaser builds the `changes` binary for Linux, macOS, and Windows.
4. GoReleaser publishes a GitHub Release and updates the internal Homebrew cask tap.

## GitHub dry run

Use the `release-dry-run` workflow from the GitHub Actions UI when you want to validate the release path without publishing a release or updating the Homebrew tap.

The dry run workflow:

- prepares release notes with `./scripts/prepare-release-notes.sh`
- validates `.goreleaser.yaml` with `./scripts/verify-release-config.sh`
- runs `./scripts/build-release-snapshot.sh`
- uploads the generated release notes and snapshot build outputs as workflow artifacts

## Local verification

Use these commands before pushing a release tag:

```bash
./scripts/prepare-release-notes.sh
./scripts/verify-release-config.sh
./scripts/build-release-snapshot.sh
```

`./scripts/verify-release-config.sh` requires GoReleaser `v2.10` or newer because this repo uses `homebrew_casks` in `.goreleaser.yaml`.
`./scripts/build-release-snapshot.sh` supplies repo-local Go cache paths and placeholder release env values so a local snapshot build can run without production release credentials.

## Release auth paths

Use one of these two paths intentionally:

- Human local rehearsal: inject release credentials from your shell, from a file-backed secret path, or from an external launcher or wrapper before running the local helper scripts.
- Automation publishing: let GitHub Actions inject repository variables and secrets on the runner side.

This repo does not treat a human desktop-auth session as the automation path, and it does not expose provider-specific secret-manager commands as part of the `changes` CLI surface.

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

## Local secret contract

The local release-helper scripts accept the tap credential through either of these inputs:

- `CHANGES_HOMEBREW_TAP_TOKEN`
- `CHANGES_HOMEBREW_TAP_TOKEN_FILE`

If both are set, the scripts fail fast and ask you to choose one.

The scripts do not accept raw secret values on argv. If you use 1Password or another secret manager locally, keep that provider-specific resolution in the launcher or wrapper that starts the script rather than in the `changes` tool core.

This repository does not currently keep checked-in provider-mapping launcher profiles, and the release helpers do not read dotenv files directly.

## Notes generation

Release automation expects the release commit to already contain:

```bash
.local/share/changes/releases/changes-<version>+github_release.md
```

For example, tag `v0.1.0` must have:

```bash
.local/share/changes/releases/changes-0.1.0+github_release.md
```

The release workflow validates that the committed file exists and passes it directly to GoReleaser. It does not regenerate release notes in CI, because mutating the checkout causes GoReleaser to fail its dirty-tree validation.

Generate or refresh the committed file locally with:

```bash
./scripts/prepare-release-notes.sh "v0.1.0"
```

The dry-run workflow uses the same release-notes preparation path.

## Tap model

The cask config is intentionally template-driven and uses placeholders for:

- the release repository owner/name
- the tap repository owner/name/branch
- the tap repository token
- the release homepage URL

This keeps the release process portable without hardcoding organization-specific values into `.goreleaser.yaml`.
