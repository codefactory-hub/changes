# `changes`

`changes` is a fragment-driven changelog and release-notes CLI for Git repositories. It is inspired by Changesets in spirit, but uses repo-local XDG-style directories, durable fragments, and per-release records instead of destructive fragment deletion.

`changes` is fragment-centric. External changelog formats are views generated from fragments plus release records; they are not the source of truth.

Default fragment shape:

```md
+++
bump = "minor"
public_api = "add"
behavior = "new"
+++

Any Markdown content goes here.
```

## Repository-local layout

Committed:

- `.config/changes/config.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/releases/`
- `.local/share/changes/templates/`

Transient:

- `.local/state/`

The tool always resolves the target repository root from Git. If a command runs outside a Git repository, it fails cleanly.

## Current command surface

```text
changes init
changes create patch --behavior fix "Fix release note rendering."
changes create minor --edit
changes status
changes version next [--pre rc]
changes release [--pre rc] [--version ...]
changes resolve --product cli --version 1.2.0 [--format json] [--output path]
changes render --version 1.2.0-rc.1 [--profile github_release] [--output path]
changes render profiles
changes changelog rebuild [--output path]
```

Interactive authoring prompts for optional `name` stem and body text when you run `create` in a TTY. Use `--edit` when the body needs richer Markdown than a single prompt line.

## Fragment vocabulary

Fragments still carry an explicit `bump` because the current release suggestion layer consumes `patch|minor|major` directly. The newer semantic levers exist to explain why that bump was chosen and to make future automation more defensible.

Use these fragment keys when they help:

- `public_api = "add|change|remove"`
- `behavior = "new|fix|redefine"`
- `dependency = "refresh|relax|restrict"`
- `runtime = "expand|reduce"`

The intended meaning is:

- `public_api`
  Additive public surface change, breaking public surface change, or public surface removal.
- `behavior`
  New observable behavior, a bug fix that better matches the prior contract, or a semantic redefinition of existing usage.
- `dependency`
  Exact lockfile-style dependency refresh without changing declared version windows, broader declared compatibility, or narrower declared compatibility.
- `runtime`
  Broader or narrower declared support for runtimes, toolchains, SDKs, deployment targets, or supported execution environments.

`type = "added|changed|fixed"` remains available as an optional render grouping for release-note sections. It is no longer the primary way the tool describes semver intent to developers.

## Model

- Fragments are durable source records. They are not deleted when a release happens.
- Release records are canonical per-release files stored under `.local/share/changes/releases/`.
- Every release identity requires one base `ReleaseRecord` named `<product>-<version>.toml`.
- Optional companion `ReleaseRecord`s use SemVer build metadata, such as `<product>-1.2.3+docs.1.toml`, for additional canonical records tied to the exact same release.
- Base release records carry lineage, fragment selection, and release-wide structure such as sections and display fields.
- `ReleaseBundle` is the assembled factual data for one release: base record, companion records, lineage context, selected fragments, and ordered sections.
- Final releases form the canonical parent-linked lineage used for changelog rebuilds.
- Prereleases are ordinary SemVer prereleases such as `alpha`, `beta`, `rc`, or any other valid label.
- A later prerelease with the same label excludes fragments already reachable from its own same-label parent chain.
- Changing prerelease labels starts a fresh prerelease lineage for the same target version.
- A final release recomputes from the previous final head, not from prerelease history.
- Build metadata groups companion records for the same release identity and never affects precedence.

## Semver behavior in the first layer

- If no stable release exists, `project.initial_version` is the first stable baseline.
- Unreleased fragments not reachable from the latest stable head determine the highest pending bump.
- Stable suggestion uses `major > minor > patch` precedence.
- Prerelease suggestion targets the next final version and increments the prerelease number within the same target version and label.

The semantic levers above typically imply bumps like this:

- `public_api = "remove"` or `public_api = "change"`: usually `major`
- `public_api = "add"`: usually `minor`
- `behavior = "redefine"`: usually `major`
- `behavior = "new"`: usually `minor`
- `behavior = "fix"` by itself: usually `patch`
- `dependency = "restrict"`: usually `major`
- `dependency = "relax"`: usually `minor`
- `dependency = "refresh"` by itself: usually no published-package bump signal
- `runtime = "reduce"`: usually `major`
- `runtime = "expand"`: usually `minor`

When a fragment carries multiple levers, the highest-severity implication should win. A `fix` combined with a `restrict`, for example, should still be treated as a likely `major`.

## Rendering

- Render behavior is configured through named template packs in `.config/changes/config.toml`.
- The built-in packs are `repository_markdown`, `github_release`, `tester_summary`, `debian_changelog`, and `rpm_changelog`.
- Single-release packs render only the selected `ReleaseBundle`.
- Chain-style packs walk `parent_version` backward from the chosen base release record and render each assembled bundle in the lineage.
- Multi-release trimming drops whole release blocks from the tail of the rendered chain. It never truncates inside an entry body.
- Repo-local template files override the built-in pack templates without changing release-record semantics.

## Development-only Collection

Upstream changelog collection is a development-only workflow. It is compiled only when you opt into the `devtools` build tag, and it is not part of the distributed `changes` binary.

- Input is a TOML catalog of remote changelog sources.
- Raw responses and normalized text snapshots are written under `.local/state/changes/collections/<timestamp>/`.
- Output can be rendered as Markdown or JSON for inspection and downstream processing.
- Invoke it with `go run -tags devtools ./cmd/changes collect --catalog .local/state/catalog.toml`.
- Or use the repo-local wrapper: `./scripts/collect-changelogs --catalog .local/state/catalog.toml`.
- To turn a collected snapshot into fragment files in ignored state storage, run `go run -tags devtools ./cmd/changes collect drafts --input .local/state/changes/catalog-check.json`.
- The extractor attempts to split each upstream changelog into release/version sections and write one fragment per extracted section.
- Extracted fragments are written per product under `.local/state/collect-changes/<product>/changes/fragments/`.
- Those `collect-changes` workspaces may also contain sibling `changes/releases/` and `changes/templates/` directories, but they are separate from the canonical `.local/share/changes/*` tree.
- Imported collection output must not be copied into `.local/share/changes/fragments`.

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
