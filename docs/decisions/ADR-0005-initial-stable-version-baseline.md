# ADR-0005 Initial Stable Version Baseline

## Status

Accepted

## Context

Version recommendation needs a deterministic stable baseline even before the repository has published its first stable release through `changes`.

Inventing a synthetic earlier release would complicate the first layer and blur the distinction between explicit release history and inferred history.

## Decision

If the repository has no stable base release records, treat `project.initial_version` as the first stable baseline.

After the first stable release exists, compute future stable targets by applying the highest unreleased bump to the latest stable version in the stable lineage.

## Consequences

- first-release version recommendation is deterministic without synthetic history
- the first layer stays simple and explicit
- some advanced historical import and migration cases remain future work
