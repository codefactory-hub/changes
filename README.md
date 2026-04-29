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

## Layout and initialization

The tool always resolves the target repository root from Git. If a command runs outside a Git repository, repo-scoped commands fail cleanly.

### Default repo layout

Plain `changes init` uses the repo-local `xdg` layout unless an approved higher-precedence default says otherwise.

Committed:

- `.config/changes/config.toml`
- `.config/changes/layout.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/releases/`
- `.local/share/changes/prompts/`

Transient:

- `.local/state/changes/`

Repo-local `xdg` init keeps `/.local/state/changes/` ignored in `.gitignore`.

Canonical example:

```bash
changes init
```

### Repo `home` layout

Use `home` when you want one repo-local root that contains config, data, and state.

Committed:

- `.changes/config/config.toml`
- `.changes/config/layout.toml`
- `.changes/data/fragments/`
- `.changes/data/releases/`
- `.changes/data/prompts/`

Transient:

- `.changes/state/`

Repo-local `home` init keeps `/.changes/state/` ignored in `.gitignore`.

Canonical example:

```bash
changes init --layout home
```

### Global layout

Plain `changes init global` creates the global `xdg` layout unless flags or `CHANGES_HOME` select the `home` variant instead.

Use global init when you want to establish global defaults and a global authoritative layout manifest. Global `xdg` uses the XDG config, data, and state directories for `changes`. Global `home` uses one root with `config`, `data`, and `state` subdirectories.

Canonical example:

```bash
changes init global --layout home
```

### Init command surface

```text
changes init [--layout xdg|home] [--home PATH]
changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]
changes init global [--layout xdg|home] [--home PATH]
```

`--home` is only valid with `--layout home`.

Successful init reports the selected layout and the resolved config, data, and state paths. Repo init mentions `.gitignore` only when it actually added or updated the authoritative state-ignore entry.

`changes init` can also bootstrap an already-released product:

- `changes init --current-version unreleased` starts a new repository with no adoption release record
- `changes init --current-version 0.0.0` is treated the same as `unreleased`
- `changes init --current-version 2.7.4` creates a standard adoption release and fragment at `2.7.4`
- when init creates an adoption release, it also generates the repo-local release-history import prompt as a starting point for an LLM-assisted historical import workflow
- rerunning `init` after bootstrap adoption artifacts already exist fails and asks you to review or remove them intentionally

### Layout selection precedence

| Workflow | Selection order |
|---|---|
| Global bootstrap | Explicit `changes init global` flags > `CHANGES_HOME` > XDG environment variables > built-in default locations |
| Repo initialization | Explicit `changes init` flags > global config `[repo.init]` defaults > `CHANGES_HOME` as a repo-style preference signal > XDG environment variables as a repo-style preference signal > built-in default locations |

### Inspecting layout state

Use `doctor` to inspect layout resolution, authority warnings, and migration guidance.

```text
changes doctor [--scope global|repo|all] [--explain] [--json]
changes doctor --scope repo --repair
changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]
```

Canonical examples:

```bash
changes init
changes init --layout home
changes init global --layout home
changes doctor --scope repo --explain
changes doctor --scope repo --repair
changes doctor --scope global
changes doctor --migration-prompt --scope repo --to home
```

By default, `changes doctor` inspects the repo scope when you are inside a repository. Use `--scope global` or `--scope all` when you need broader inspection. `--migration-prompt` prints the advisory Markdown brief to stdout unless you supply `--output PATH`.

Existing repos initialized by current `changes` flows include `layout.toml` and continue to operate normally. Older repos that only have legacy directory/config shapes without `layout.toml` are treated as legacy layouts.

Use `changes doctor --scope repo --repair` when exactly one repo-local legacy layout already contains the real config/data/state directories and only needs its authoritative manifest restored. Repair stamps `layout.toml`, preserves the authoritative repo-local state ignore rule, and does not move data.

Use `changes doctor --migration-prompt --scope repo --to home` or the corresponding `xdg` target when the repo has conflicting repo-local layouts, when you need to relocate data, or when you want explicit migration guidance before changing on-disk state.

## Current command surface

```text
changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]
changes init global [--layout xdg|home] [--home PATH]
changes create --behavior fix "Fix release note rendering."
changes create --public-api add --edit
changes status
changes status --explain
changes status --json
changes doctor [--scope global|repo|all] [--explain] [--json]
changes doctor --scope repo --repair
changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]
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
changes render profiles --json
```

Interactive authoring prompts for optional `name` stem and body text when you run `create` in a TTY. Use `--edit` when the body needs richer Markdown than a single prompt line.

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
- If no final base release record exists, the repository's current version is still `unreleased`.
- Before the first final release record exists, `project.initial_version` is only the deterministic initial final-release target.
- Unreleased fragments not reachable from the latest final head determine the recommended bump through semantic levers plus `versioning.public_api`.
- Prerelease suggestion targets the next final version and increments the prerelease number within the same target version and label.
- Prerelease labels are explicit per release command, such as `changes release --accept --pre beta`; there is no configured default label.

Repositories that adopt `changes` mid-lifecycle usually move onto an explicit release-record baseline immediately through `changes init --current-version <semver>`. Brand-new repositories that initialize with `unreleased` continue to use `project.initial_version` only as their initial final-release target until their first final release record exists.

Configure the public API policy in `.config/changes/config.toml`:

```toml
[project]
initial_version = "0.1.0"

[versioning]
public_api = "unstable"
```

`project.initial_version` is a deterministic initial final-release target, not an always-updated current-version field.

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

That policy drives the tool's recommendation. `changes status --explain` and interactive `changes release` show the recommended bump and the evidence behind it. Use `changes status --json` for structured status output in scripts and agent workflows. Non-interactive release requires an explicit decision: `changes release --accept` to accept the recommendation, or `changes release --override --bump <patch|minor|major>` / `changes release --override --version <exact>` to override it.

## Rendering

- `render` is the public output command.
- Render behavior is configured through named render profiles in `.config/changes/config.toml`.
- Each render profile resolves to a concrete template pack at render time.
- The built-in render profiles are `repository_markdown`, `github_release`, `tester_summary`, `debian_changelog`, and `rpm_changelog`.
- Use `changes render profiles --json` for structured profile metadata in scripts and agent workflows.
- `changes render --latest --profile repository_markdown` is the public path for rebuilding `CHANGELOG.md`.
- Single-release profiles render only the selected `ReleaseBundle`.
- Chain-style profiles walk `parent_version` backward from the chosen base release record and render each assembled bundle in the lineage.
- Multi-release trimming drops whole release blocks from the tail of the rendered chain. It never truncates inside an entry body.
- Repo-local template files, when present, override the built-in profile templates without changing release-record semantics.

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

## Governance Lifecycle

This repository keeps durable governance and process policy in repo-local documents rather than in external agent instructions. During `repository-governance` audit and remediation work, the skill's canonical governance semantics govern ambiguity resolution unless this repository has an explicit accepted compatible local exception.

- `docs/01-ideas/` stores raw, intentionally lightweight thoughts and seeds.
- `docs/02-research/` stores forward-looking investigation, option comparison, uncertainty reduction, and external research.
- `docs/03-decisions/` stores durable decisions, typically ADRs.
- `docs/04-plans/` stores implementation intent for work being done now or about to begin now.
- `docs/05-insights/` stores backward-looking learning, debugging takeaways, operational lessons, and retrospectives.
- `docs/99-archive/` stores terminal-state artifacts and preserves them under matching prefixed archive subfolders.

Document routing is strict:

- ADR = what we decided.
- Plan = what we are doing right now.
- Ideas are early and lightweight.
- Research reduces uncertainty before a decision.
- Insights preserve what execution or operations taught us.

Archived docs never return to live folders. When moving a governed doc into the archive, set `status: archived`, move it under the matching archive folder, and do not modify it afterward. New work creates a new live doc that references the archived predecessor.

All governed docs include `created: YYYY-MM-DD`. ADRs also include `status: proposed | accepted | superseded`.

Governed docs are Markdown documents and may include YAML frontmatter wherever the governance contract requires metadata.

In user-facing governed prose, reference other repo-authored Markdown docs with inline Markdown links that use relative URLs and human-readable link text. Prefer the destination H1 when it reads naturally, and otherwise use a concise prose variant that stays clear in sentence context.

Prefer prose-embedded links over raw backticked paths or path-only code blocks when pointing readers to another governed doc.

Keep raw code formatting for literal filenames, path patterns, shell commands, inventories, and non-user-facing internal reference material.

Plans may optionally include `governance_audit: remediation-plan` when they establish the repository's post-audit governance baseline. Future governance audits should treat the newest such plan as the cutoff for historical-format drift in older plans, insights, and similar closed execution records, but not for live governing docs.

Use these filename patterns:

- `idea-{slug}.md`
- `RPT-YYYY-MM-DD-NN-{slug}.md`
- `ADR-NNNN-{slug}.md`
- `PLN-YYYY-MM-DD-NN-{slug}.md`
- `INS-YYYY-MM-DD-NN-{slug}.md`

For research, plans, and insights, `NN` is the zero-padded same-day sequence for that document kind. When upgrading an existing repository to this rule, derive same-day sequence from the best available creation signal in this order: explicit timestamp metadata, filesystem creation time, filesystem modification time, then stable lexical filename order.

Use one slug recipe across all governed docs: strip diacritics, lowercase, replace `&` with `and`, remove apostrophes, remove stop words, replace remaining non-alphanumerics with `-`, collapse repeated `-`, and trim leading and trailing `-`.

Pre-baseline historical research and planning records are preserved conservatively after the governance migration recorded in [PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns](docs/04-plans/PLN-2026-04-21-01-modernize-governance-and-adopt-relevant-toolsmith-patterns.md). They now live in canonical folders, but they may retain legacy filenames or lighter metadata when preserving their original history is more accurate than retrofitting them.

Operational guides that do not fit the governed lifecycle, such as [Releasing `changes`](docs/guides/RELEASING.md), live under `docs/guides/`.

## Current ADRs

- [ADR-0001: Use repo-local XDG layout](docs/03-decisions/ADR-0001-repo-local-xdg-layout.md)
- [ADR-0002: Keep fragment artifacts durable](docs/03-decisions/ADR-0002-durable-fragment-artifacts.md)
- [ADR-0003: Represent releases as parent-linked records](docs/03-decisions/ADR-0003-parent-linked-release-manifests.md)
- [ADR-0004: Treat rendered outputs as views](docs/03-decisions/ADR-0004-rendered-outputs-are-views.md)
- [ADR-0005: Establish the initial stable version baseline](docs/03-decisions/ADR-0005-initial-stable-version-baseline.md)
- [ADR-0006: Use Homebrew cask distribution for internal releases](docs/03-decisions/ADR-0006-homebrew-cask-distribution.md)
- [ADR-0007: Adopt lightweight repository governance lifecycle](docs/03-decisions/ADR-0007-adopt-lightweight-repository-governance-lifecycle.md)
- [ADR-0008: Separate human release auth from automation and agent auth](docs/03-decisions/ADR-0008-separate-human-release-auth-from-automation-and-agent-auth.md)
- [ADR-0009: Adopt provider-neutral release secret ingestion](docs/03-decisions/ADR-0009-adopt-provider-neutral-release-secret-ingestion.md)

## Planning And Implementation

- Before starting implementation, check whether the working directory is clean.
- If the working directory is not clean, pause and tell the user.
- Do not begin implementation until the user explicitly chooses one of these three paths:
  - checkpoint the current changes
  - treat the current changes as part of the approved work
  - proceed intentionally with a dirty working directory
- If the active runtime provides a structured human-response or multiple-choice UI, prefer that UI for the same three dirty-worktree choices. Otherwise, ask the same three dirty-worktree choices in plain text.
- If the active runtime supports a planning stage with structured human approval, perform the dirty-worktree decision during that planning stage when practical.
- If the user chooses to checkpoint the current changes, use the `git-workflow` skill to create cohesive commits until the working directory is clean.
- Host planning and written repository plans are not the same thing.
- A written repository plan is required before implementation for non-trivial changes.
- A written repository plan is not required for trivial changes.
- Treat a change as non-trivial when it materially affects user-visible behavior or UI, introduces or materially changes a public interface or command surface, spans multiple files or subsystems in a way that benefits from sequencing, introduces migration or rollback concerns, or otherwise requires documented implementation intent.
- Treat a change as trivial when it is narrowly scoped and does not materially change user-visible behavior, public interfaces, or implementation scope.
- If the threshold is unclear, ask whether the work should be treated as trivial or non-trivial rather than defaulting to a written repository plan.
- Planning in the host does not by itself require saving a written repository plan.
- When the agent determines that work is trivial under this policy, it should say so and proceed without creating a durable plan record.
- When a written repository plan is required, the first implementation step must be to save that plan as a Markdown document under `docs/04-plans/` using the naming rules for plans.
- After the plan file is saved, implementation work may begin.
- Every written repository plan must direct the agent to use the `git-workflow` skill as often as prudent to create semantically cohesive commits during implementation.
- A written repository plan may be completed in one commit or in multiple commits, depending on the natural shape of the work.
- A written repository plan is complete when its intended implementation and required validation are finished, the completed work has been committed, and the working directory is clean.
- Git actions must run in series and never in parallel because concurrent Git operations create avoidable lock conflicts and unnecessary user-facing failures.
- The last implementation step in every written repository plan must be to use the `git-workflow` skill to finish committing the work completed for that plan and leave the working directory clean.
- After a written repository plan is complete, additional trivial follow-up work may proceed without reopening the completed plan or creating a new plan.
- Create a new written repository plan for follow-up work only when that follow-up work is itself non-trivial.

## Development

This repo is intentionally bootstrapped with a modest standard-library-first CLI and a single TOML dependency.

Useful local commands:

```bash
go test ./...
```

## Release automation

Release automation is wired through GoReleaser and a private/internal Homebrew cask tap. Use `./scripts/prepare-release-notes.sh`, `./scripts/verify-release-config.sh`, and `./scripts/build-release-snapshot.sh` for local release verification, and see [Releasing `changes`](docs/guides/RELEASING.md) for the full publish flow, the release-auth split between human and automation paths, and the local secret-input contract.

For CI rehearsal without publishing, use the `release-dry-run` GitHub Actions workflow.
