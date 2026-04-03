# ADR-0004 Rendered Outputs Are Views

## Status

Accepted

## Context

`changes` must render multiple downstream formats such as repository changelogs, GitHub release bodies, tester summaries, and package-manager changelog text.

If those output formats affect fragment selection or become alternate durable sources of truth, the system stops being fragment-centric and becomes difficult to reason about.

## Decision

Keep `changes` fragment-centric.

External changelog formats are render-time views generated from canonical fragment and release-record data. Built-in template packs may target repository Markdown, release bodies, or package-manager-specific formats, but those outputs do not influence selection semantics or release lineage.

Rendering policy belongs to template packs and render configuration, not to release records.

## Consequences

- one selection model can feed many output formats
- output-specific formatting does not distort the release data model
- repos can customize rendering without changing release semantics
