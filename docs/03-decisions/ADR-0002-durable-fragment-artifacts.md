---
created: 2026-04-02
status: accepted
---

# ADR-0002: Keep fragment artifacts durable

## Status

Accepted

## Context

Fragment-driven release-note systems often delete or mutate fragment files as part of a release. That makes preview lines awkward, weakens auditability, and makes it harder to regenerate or repair historical release notes after templates or policies change.

`changes` is intended to preserve durable, human-editable source records for release-note content.

## Decision

Treat fragment files as durable source artifacts.

Releases do not delete fragments and do not rewrite fragment content as part of normal release creation.

Fragments remain the authoring unit for change entries, while base release records record which fragments were added to a given release-line step.

## Consequences

- release-note source records remain auditable after release creation
- historical changelog sections can be regenerated from fragments plus release records
- preview and stable releases can reference the same underlying fragment set without destructive mutation
