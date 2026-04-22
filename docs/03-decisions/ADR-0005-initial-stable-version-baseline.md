---
created: 2026-04-02
status: accepted
---

# ADR-0005: Establish the initial stable version baseline

## Status

Accepted

## Context

Version recommendation needs a deterministic initial final-release target even before the repository has published its first final release through `changes`.

Some repositories adopt `changes` after they already have a released version. Those repositories need an explicit bootstrap path so the current version becomes part of the release-record lineage instead of living forever as ad hoc config state.

At the same time, brand-new repositories still need a deterministic initial final-release target before any final release record exists.

## Decision

- If a repository adopts `changes` at an already released version, `changes init --current-version <semver>` creates an explicit bootstrap adoption release record at that version.
- Once any final base release record exists, including a bootstrap adoption release, use the latest final base release record as the current-version baseline.
- If the repository has no final base release records, treat the current version as `unreleased` and treat `project.initial_version` as the deterministic initial final-release target.

After a final base release exists, compute future final targets by applying the policy-derived recommended bump to the latest final version in the final lineage.

## Consequences

- new repositories still get deterministic first-release recommendation without synthetic history
- adopted repositories can move onto explicit release-record history immediately
- `project.initial_version` remains an initial target seed, not a perpetually current version field
- richer historical reconstruction beyond the standard adoption bootstrap remains future work
