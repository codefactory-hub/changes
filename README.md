# `changes`

`changes` is a language-agnostic changelog and release-notes tool for Git repositories.

It helps developers record release-relevant changes in simple terms at the time they make the change, without forcing them to decide the release version immediately. Later, when a team prepares a release, `changes` uses those recorded change descriptions together with the repository's versioning policy to suggest the next version and create an explicit release record. From that recorded release history, it can render outputs such as a repository changelog or release-note text for a publishing surface like GitHub or GitLab.

`changes` does not treat Git commits as release history.

Git commits are optimized for building software: they capture implementation steps, review iterations, refactors, and other source-control details. Release history has a different job. It needs to surface the changes that matter for versioning, changelogs, and release communication. That is why `changes` uses fragments instead of commit history directly. A pull request might produce no fragments, one fragment, or several, depending on what is worth carrying forward into release history.

The tool keeps three concerns separate:

- **Describing changes.** Fragments record individual release-relevant changes close to when the work happens.
- **Deciding releases.** Release records capture the decision to cut a specific release at a specific version and define the release lineage over time.
- **Communicating releases.** Rendered outputs such as `CHANGELOG.md` or release-body text are generated views built from fragments plus release records.

That separation lets teams describe work as they go, make release decisions later with better context, and publish different output formats without rewriting the underlying history.

## How it works

1. Developers create fragments as they work.
2. At release time, `changes` reviews the pending fragments and recommends a version.
3. A release record is created to capture that release decision.
4. Changelogs and release-note documents are rendered from the recorded release history.

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
- `.local/share/changes/releases/`
- `.local/share/changes/prompts/`
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
changes release --accept
changes release --accept --pre rc
changes release --override --bump minor
changes release --override --bump minor --pre rc
changes release --override --version 1.2.0
changes release --override --version 1.2.0-rc.3
changes render --version 1.2.0-rc.1 [--profile github_release] [--output path]
changes render --latest --profile repository_markdown > CHANGELOG.md
changes render profiles
```

Interactive authoring prompts for optional `name` stem and body text when you run `create` in a TTY. Use `--edit` when the body needs richer Markdown than a single prompt line.

`changes init` can also bootstrap an already-released product:

- `changes init --current-version unreleased` starts a new repository with no adoption release record
- `changes init --current-version 0.0.0` is treated the same as `unreleased`
- `changes init --current-version 2.7.4` creates a standard adoption release and fragment at `2.7.4`
- when init creates an adoption release, it also generates `.local/share/changes/prompts/release-history-import-llm-prompt.md` as a repo-specific starting point for an LLM-assisted historical import workflow
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

Inspect the derived impact evidence with `changes status --explain`. In a TTY, `changes release` shows the same evidence, proposes a default release version when one can be inferred, and lets a human accept the recommendation with Enter or override it with `patch`, `minor`, or `major`. If no version bump is inferred from the fragment levers, `release` requires an explicit override choice.

## Model

- Fragments are durable source records. They are not deleted when a release happens.
- Release records are canonical per-release files stored under `.local/share/changes/releases/`.
- Prompt files under `.local/share/changes/prompts/` are optional repo-specific helper artifacts created during adoption bootstrap. They are not canonical release history.
- Every release identity requires one base `ReleaseRecord` named `<product>-<version>.toml`.
- Optional companion `ReleaseRecord`s use SemVer build metadata, such as `<product>-1.2.3+docs.1.toml`, for additional canonical records tied to the exact same release.
- Base release records carry lineage, fragment selection, and release-wide structure such as sections and display fields.
- Init can create a standard bootstrap adoption release and fragment for an already-released product. Those artifacts are ordinary renderable history and establish the current-version baseline for later `status` and `release` calculations.
- `ReleaseBundle` is the assembled factual data for one release: base record, companion records, lineage context, selected fragments, and ordered sections.
- Final releases form the canonical parent-linked lineage used for repository changelog rendering.
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
- Prerelease labels are explicit per release command, such as `changes release --accept --pre beta`; there is no configured default label.

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

When `changes init` bootstraps an already-released product, it generates `.local/share/changes/prompts/release-history-import-llm-prompt.md`.

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

That policy drives the tool's recommendation. `changes status --explain` and interactive `changes release` show the recommended bump and the evidence behind it. Non-interactive release requires an explicit decision: `changes release --accept` to accept the recommendation, or `changes release --override --bump <patch|minor|major>` / `changes release --override --version <exact>` to override it.

## Rendering

- `render` is the public output command.
- Render behavior is configured through named render profiles in `.config/changes/config.toml`.
- Each render profile resolves to a concrete template pack at render time.
- The built-in render profiles are `repository_markdown`, `github_release`, `tester_summary`, `debian_changelog`, and `rpm_changelog`.
- `changes render --latest --profile repository_markdown` is the public path for rebuilding `CHANGELOG.md`.
- Single-release profiles render only the selected `ReleaseBundle`.
- Chain-style profiles walk `parent_version` backward from the chosen base release record and render each assembled bundle in the lineage.
- Multi-release trimming drops whole release blocks from the tail of the rendered chain. It never truncates inside an entry body.
- Repo-local template files override the built-in profile templates without changing release-record semantics.

## Further Reading

These references are useful background for how `changes` thinks about compatibility, release impact, and why “breaking change” is not always reducible to a single SemVer label.

- [Semantic Versioning](https://semver.org/) for the baseline versioning contract most ecosystems start from.
- [Dart package versioning](https://dart.dev/tools/pub/versioning) for a concrete explanation of constraint solving, lockfiles, exported dependencies, and why dependency compatibility is broader than a package's own direct API.
- [Dart language versioning](https://dart.dev/language/versioning) for an example of language-level breaking changes that are managed outside ordinary package SemVer.
- [Swift PackageDescription](https://docs.swift.org/package-manager/PackageDescription/PackageDescription.html) for `swift-tools-version`, platform support, and package-level compatibility declarations.
- [Library Evolution in Swift](https://www.swift.org/blog/library-evolution/) for source and binary compatibility tradeoffs, including why additive API changes such as enum cases can still be breaking in some client contexts.
- [PubGrub incompatibilities](https://pubgrub-rs-guide.netlify.app/internals/incompatibilities) for the underlying conflict model behind modern dependency solving.
- [Dependency Resolution Made Simple](https://borretti.me/article/dependency-resolution-made-simple) for a practical explanation of why version constraints and selected versions are different things.
- [Categorizing Package Manager Clients](https://nesbitt.io/2025/12/29/categorizing-package-manager-clients.html) and [Dependency Resolution Methods](https://nesbitt.io/2026/02/06/dependency-resolution-methods.html) for a cross-ecosystem view of solver behavior, nesting, mediation, and why runtime compatibility depends on more than SemVer labels alone.

## Development

This repo is intentionally bootstrapped with a modest standard-library-first CLI and a single TOML dependency.

Useful local commands:

```bash
go test ./...
```

## Release automation

Release automation is wired through GoReleaser and a private/internal Homebrew cask tap. Use `./scripts/prepare-release-notes.sh`, `./scripts/verify-release-config.sh`, and `./scripts/build-release-snapshot.sh` for local release verification, and see [docs/releasing/RELEASING.md](docs/releasing/RELEASING.md) for required variables, secrets, and the full publish flow.

For CI rehearsal without publishing, use the `release-dry-run` GitHub Actions workflow.
