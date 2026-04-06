# ADR-0005 Initial Stable Version Baseline

## Status

Accepted

## Context

Version recommendation needs a deterministic stable baseline even before the repository has published its first final release through `changes`.

Some repositories adopt `changes` after they already have a released version. Those repositories need an explicit bootstrap path so the current version becomes part of the release-record lineage instead of living forever as ad hoc config state.

At the same time, brand-new repositories still need a deterministic fallback baseline before any final release record exists.

## Decision

- If a repository adopts `changes` at an already released version, `changes init --current-version <semver>` creates an explicit bootstrap adoption release record at that version.
- Once any final base release record exists, including a bootstrap adoption release, use the latest final base release record as the current-version baseline.
- If the repository has no final base release records, treat `project.initial_version` as the deterministic first stable baseline.

After a final base release exists, compute future stable targets by applying the policy-derived recommended bump to the latest final version in the final lineage.

## Consequences

- new repositories still get deterministic first-release recommendation without synthetic history
- adopted repositories can move onto explicit release-record history immediately
- `project.initial_version` remains a fallback seed, not a perpetually current version field
- richer historical reconstruction beyond the standard adoption bootstrap remains future work
