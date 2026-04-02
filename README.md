# `changes`

`changes` is a fragment-driven changelog and release-notes CLI for Git repositories. It is inspired by Changesets in spirit, but uses repo-local XDG-style directories, durable fragments, and per-release manifests instead of destructive fragment deletion.

## Repository-local layout

Committed:

- `.config/changes/config.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/releases/`
- `.local/share/changes/templates/`

Transient:

- `.local/state/changes/`

The tool always resolves the target repository root from Git. If a command runs outside a Git repository, it fails cleanly.

## Current command surface

```text
changes init
changes add --title ... --type fixed --bump patch --body ...
changes status
changes version next [--pre rc]
changes release create [--channel preview|stable] [--pre rc] [--version ...]
changes render --version 1.2.0-rc.1 [--output path]
changes changelog rebuild [--output path]
```

## Model

- Fragments are durable source records. They are not deleted when a release happens.
- Release manifests freeze the fragment IDs chosen for a specific release.
- Preview manifests do not consume fragments globally.
- Stable manifests consume fragments logically by reference, which keeps historical rebuilds possible.
- Preview release lines consume within their own line only. `1.2.0-rc.2` does not repeat what `1.2.0-rc.1` already referenced, while `1.2.0-beta.1` starts a fresh line and can show the same stable-unreleased fragments again.

## Semver behavior in the first layer

- If no stable release exists, `project.initial_version` is the first stable baseline.
- Unreleased fragments not referenced by any consuming stable manifest determine the highest pending bump.
- Stable suggestion uses `major > minor > patch` precedence.
- Preview suggestion targets the next stable version and increments the prerelease number within the same release line.

## Limit handling

Rendering enforces `render.max_chars` by dropping whole rendered entries from the bottom of the ordered entry list. It never truncates inside an entry body. When dropping occurs, the configured omission notice is appended.

## Development

This repo is intentionally bootstrapped with a modest standard-library-first CLI and a single TOML dependency.

Useful local commands:

```bash
env GOCACHE=$PWD/.local/state/changes/gocache \
  GOPATH=$PWD/.local/state/changes/gopath \
  GOMODCACHE=$PWD/.local/state/changes/gomodcache \
  go test ./...
```

## Release automation

Release automation is wired through GoReleaser and a private/internal Homebrew cask tap. See [docs/releasing/RELEASING.md](docs/releasing/RELEASING.md) for required variables, secrets, and the intended release flow.
