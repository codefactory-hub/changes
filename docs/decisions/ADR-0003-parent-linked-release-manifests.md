# ADR-0003 Parent-linked Release Manifests

## Status

Accepted

## Context

`changes` needs to support preview and stable releases, multiple releases in a day, and non-destructive fragment reuse across release lines.

A destructive “consumed fragment” flag does not model preview history cleanly because consumption is line-relative. What matters is which fragments have already been introduced within a given release lineage.

## Decision

Represent each release as an append-only manifest that records:

- the emitted `version`
- the `target_version`
- the `channel`
- the immediate `parent_version` within that release line
- the `added_fragment_ids` newly introduced by that release record

Preview releases form their own parent-linked lineages. Stable releases form a separate stable lineage.

Fragments reachable from the latest stable head are no longer globally unreleased for future stable recommendations. Preview lines only exclude fragments already reachable from their own preview ancestry.

## Consequences

- preview and stable release lines remain structurally separate
- fragment “consumption” is modeled by lineage reachability rather than destructive mutation
- release selection remains durable and auditable
- changelog rebuilds can walk a manifest lineage deterministically

