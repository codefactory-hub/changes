# `changes`

`changes` is a fragment-driven changelog and release-notes CLI for Git repositories. It is inspired by Changesets in spirit, but uses repo-local XDG-style directories, durable fragments, and per-release manifests instead of destructive fragment deletion.

`changes` is fragment-centric. External changelog formats are views generated from fragments plus release manifests; they are not the source of truth.

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
changes render --version 1.2.0-rc.1 [--profile github_release] [--output path]
changes render profiles
changes changelog rebuild [--output path]
```

## Model

- Fragments are durable source records. They are not deleted when a release happens.
- Release manifests are append-only selection records.
- Each manifest stores only `added_fragment_ids` for that release plus an optional `parent_version`.
- Preview and stable releases each form their own parent-linked lineages.
- A later preview in the same line excludes fragments already reachable from its preview parent chain.
- A final stable release recomputes from the previous stable head, not from the preview chain.

## Semver behavior in the first layer

- If no stable release exists, `project.initial_version` is the first stable baseline.
- Unreleased fragments not reachable from the latest stable head determine the highest pending bump.
- Stable suggestion uses `major > minor > patch` precedence.
- Preview suggestion targets the next stable version and increments the prerelease number within the same release line.

## Rendering

- Render behavior is configured through named template packs in `.config/changes/config.toml`.
- The built-in packs are `repository_markdown`, `github_release`, `tester_summary`, `debian_changelog`, and `rpm_changelog`.
- Single-release packs render only the selected release record.
- Chain-style packs walk `parent_version` backward from the chosen head.
- Multi-release trimming drops whole release blocks from the tail of the rendered chain. It never truncates inside an entry body.
- Repo-local template files override the built-in pack templates without changing manifest semantics.

## Development-only Collection

Upstream changelog collection is a development-only workflow. It is compiled only when you opt into the `devtools` build tag, and it is not part of the distributed `changes` binary.

- Input is a TOML catalog of remote changelog sources.
- Raw responses and normalized text snapshots are written under `.local/state/changes/collections/<timestamp>/`.
- Output can be rendered as Markdown or JSON for inspection and downstream processing.
- Invoke it with `go run -tags devtools ./cmd/changes collect --catalog catalog.toml`.
- Or use the repo-local wrapper: `./scripts/collect-changelogs --catalog catalog.toml`.

Example catalog:

```toml
[[sources]]
name = "Go"
url = "https://go.dev/doc/devel/release"
format = "html"

[[sources]]
name = "Node.js"
url = "https://raw.githubusercontent.com/nodejs/node/main/doc/changelogs/CHANGELOG_V22.md"
format = "markdown"
```

## Development

This repo is intentionally bootstrapped with a modest standard-library-first CLI and a single TOML dependency.

Useful local commands:

```bash
go test ./...
```

## Release automation

Release automation is wired through GoReleaser and a private/internal Homebrew cask tap. See [docs/releasing/RELEASING.md](docs/releasing/RELEASING.md) for required variables, secrets, and the intended release flow.
