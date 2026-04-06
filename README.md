# `changes`

`changes` is a fragment-driven changelog and release-notes CLI for Git repositories. It is inspired by Changesets in spirit, but uses repo-local XDG-style directories, durable fragments, and per-release records instead of destructive fragment deletion.

`changes` is fragment-centric. External changelog formats are views generated from fragments plus release records; they are not the source of truth.

Default fragment shape:

```md
+++
public_api = "add"
behavior = "new"
+++

Any Markdown content goes here.
```

## Repository-local layout

Committed:

- `.config/changes/config.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/prompts/`
- `.local/share/changes/releases/`
- `.local/share/changes/templates/`

Transient:

- `.local/state/`

The tool always resolves the target repository root from Git. If a command runs outside a Git repository, it fails cleanly.

## Current command surface

```text
changes init [--current-version <semver|unreleased>]
changes create --behavior fix "Fix release note rendering."
changes create --public-api add --edit
changes status
changes status --explain
changes release
changes release --pre rc
changes release --version 1.2.0
changes release --bump minor
changes release --yes
changes resolve --product cli --version 1.2.0 [--format json] [--output path]
changes render --version 1.2.0-rc.1 [--profile github_release] [--output path]
changes render profiles
changes changelog rebuild [--output path]
```

Interactive authoring prompts for optional `name` stem and body text when you run `create` in a TTY. Use `--edit` when the body needs richer Markdown than a single prompt line.

`changes init` can also bootstrap an already-released product:

- `changes init --current-version unreleased` starts a new repository with no adoption release record
- `changes init --current-version 0.0.0` is treated the same as `unreleased`
- `changes init --current-version 2.7.4` creates a standard adoption release and fragment at `2.7.4`
- init always generates `.local/share/changes/prompts/release-history-import-llm-prompt.md` as a repo-specific starting point for an LLM-assisted historical import workflow
- rerunning `init` after bootstrap adoption artifacts already exist fails and asks you to review or remove them intentionally

## Fragment vocabulary

Fragments describe change facts. They do not carry an explicit `patch|minor|major` bump. `changes` derives release impact from these semantic levers together with the repository's `versioning.public_api` policy.

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

Inspect the derived impact evidence with `changes status --explain`. In a TTY, `changes release` shows the same evidence, proposes a default release version when one can be inferred, and lets a human accept it with Enter or override it with `patch`, `minor`, or `major`. If no version bump is inferred from the fragment levers, `release` requires an explicit human choice.

## Model

- Fragments are durable source records. They are not deleted when a release happens.
- Release records are canonical per-release files stored under `.local/share/changes/releases/`.
- Prompt files under `.local/share/changes/prompts/` are repo-specific helper artifacts, not canonical release history.
- Every release identity requires one base `ReleaseRecord` named `<product>-<version>.toml`.
- Optional companion `ReleaseRecord`s use SemVer build metadata, such as `<product>-1.2.3+docs.1.toml`, for additional canonical records tied to the exact same release.
- Base release records carry lineage, fragment selection, and release-wide structure such as sections and display fields.
- Init can create a standard bootstrap adoption release and fragment for an already-released product. Those artifacts are ordinary renderable history and establish the current-version baseline for later `status` and `release` calculations.
- `ReleaseBundle` is the assembled factual data for one release: base record, companion records, lineage context, selected fragments, and ordered sections.
- Final releases form the canonical parent-linked lineage used for changelog rebuilds.
- Prereleases are ordinary SemVer prereleases such as `alpha`, `beta`, `rc`, or any other valid label.
- A later prerelease with the same label excludes fragments already reachable from its own same-label parent chain.
- Changing prerelease labels starts a fresh prerelease lineage for the same target version.
- A final release recomputes from the previous final head, not from prerelease history.
- Build metadata groups companion records for the same release identity and never affects precedence.

## Versioning policy

- The latest final base release record is the current version baseline when one exists.
- If no final base release record exists, `project.initial_version` remains the deterministic first stable baseline.
- Unreleased fragments not reachable from the latest final head determine the recommended bump through semantic levers plus `versioning.public_api`.
- Prerelease suggestion targets the next final version and increments the prerelease number within the same target version and label.
- Prerelease labels are explicit per release command, such as `changes release --pre beta`; there is no configured default label.

Repositories that adopt `changes` mid-lifecycle usually move onto an explicit release-record baseline immediately through `changes init --current-version <semver>`. Brand-new repositories that initialize with `unreleased` continue to rely on `project.initial_version` until their first final release record exists.

Configure the public API policy in `.config/changes/config.toml`:

```toml
[project]
initial_version = "0.1.0"

[versioning]
public_api = "unstable"
```

`project.initial_version` is a deterministic fallback baseline, not an always-updated current-version field.

## Historical Import Prompt

`changes init` always generates `.local/share/changes/prompts/release-history-import-llm-prompt.md`.

- The prompt explains the repository's `changes` layout and current bootstrap state.
- For adopted repositories, it explains that the standard adoption release and fragment may be replaced or refined intentionally.
- The CLI never invokes an LLM directly. The prompt is a human-reviewed starting point for reconstructing older history from changelogs, git history, tags, or other repo-specific evidence.

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

When a fragment carries multiple levers, the highest-severity implication should win. A `fix` combined with a `restrict`, for example, should still be treated as a likely `major`. If a fragment carries none of these levers, the tool infers no version bump from that fragment alone.

The current policy layer distinguishes between stable and unstable public APIs through `versioning.public_api`:

- `public_api = "stable"` keeps the usual SemVer interpretation, so breaking-looking levers such as `public_api = "change"`, `dependency = "restrict"`, or `runtime = "reduce"` suggest `major`
- `public_api = "unstable"` softens those same breaking-looking levers to `minor`
- additive levers such as `public_api = "add"`, `behavior = "new"`, `dependency = "relax"`, and `runtime = "expand"` still suggest `minor`
- `behavior = "fix"` still suggests `patch`

That policy drives the tool's recommendation. `changes status --explain` and interactive `changes release` show the recommended bump and the evidence behind it. `changes release --yes` accepts the default recommendation when one exists, while `changes release --bump <patch|minor|major>` or `changes release --version <exact>` lets the operator choose explicitly.

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
