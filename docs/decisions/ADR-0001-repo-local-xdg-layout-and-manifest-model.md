# ADR-0001 Repo-local XDG Layout And Manifest Model

## Status

Accepted

## Context

`changes` is intended to run inside arbitrary repositories while keeping changelog state durable, reviewable, and recoverable. Traditional destructive release-note flows make preview releases awkward and make historical repairs expensive because the source artifacts disappear once consumed.

`changes` is fragment-centric. External changelog formats are generated views over canonical fragment and manifest data rather than first-class source-of-truth files.

## Decision

### Repo-local XDG-style layout

Use repository-relative XDG-like paths:

- `.config/changes/config.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/releases/`
- `.local/share/changes/templates/`
- `.local/state/changes/`

This keeps durable state with the repository, makes review straightforward, and avoids hiding operational data in a user home directory. User-level XDG locations still exist for future shared defaults, but the primary source of truth is local to the repo being changed.

### Durable fragments

Fragments are durable source artifacts. Releases do not delete them. This preserves auditability, allows old release sections to be rebuilt later, and avoids one-way mutation.

### Release manifests

Each release writes a manifest that freezes:

- the emitted version
- the stable target version
- preview versus stable channel
- the immediate `parent_version` within that release line
- the specific `added_fragment_ids` introduced by that release record

Manifests are the durable release boundary. Rendering and changelog rebuilds derive from manifests plus fragments plus templates.

Built-in template packs may target repository Markdown, release bodies, or package-manager-style changelog text, but those outputs remain render-time views. They must not influence fragment selection or release lineage semantics.

### Preview semantics

Preview manifests do not carry a separate consumption flag. They establish line-local history through `parent_version`. A later preview in the same line excludes fragments already reachable from that line’s parent chain, while a new prerelease line starts fresh from the globally stable-unreleased fragment set.

This preserves accurate RC deltas without losing the ability to produce a final stable release from the same source fragments.

### Stable semantics

Stable manifests form their own parent-linked chain. Fragments reachable from the latest stable head are no longer globally unreleased for future stable recommendations.

### First-release semver policy

If the repository has no stable consuming manifests, `project.initial_version` is treated as the first stable target. After the first stable release exists, future targets are computed by applying the highest unreleased bump to the latest stable version.

This is intentionally conservative. It avoids inventing a synthetic pre-history just to make the initial version arithmetic work.

### Homebrew cask choice

The release automation uses a Homebrew cask path, even though CLI tooling is often discussed in formula terms elsewhere, because the delivery requirement here is an internal/private tap with binary artifacts and explicit URL ownership. The bootstrap keeps that path obvious and configurable without assuming public-source formula conventions or embedding organization-specific credentials in the repository.

## Consequences

- historical release notes can be regenerated or repaired later
- preview releases remain non-destructive
- stable consumption is explicit and auditable
- the first layer stays simple enough for maintainers to extend without undoing the data model
- some semver edge cases are intentionally deferred and documented rather than overfit early
