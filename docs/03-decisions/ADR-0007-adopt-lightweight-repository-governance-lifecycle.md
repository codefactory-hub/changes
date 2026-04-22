---
created: 2026-04-21
status: accepted
---

# ADR-0007: Adopt lightweight repository governance lifecycle

## Context

`changes` is a real tool repository with accepted local ADR history, active implementation plans, and supporting operational docs. Its governance records were created before the current canonical lifecycle model and still lived in legacy paths such as `docs/decisions/`, `docs/plans/`, and `docs/research/`.

That older layout does not make current routing, archive behavior, or planning discipline explicit enough for ongoing maintenance. The repository needs local governance in current canonical form without discarding the intent or numbering of the ADRs it already accepted.

## Decision

- Adopt the live governance lifecycle folders `docs/01-ideas/`, `docs/02-research/`, `docs/03-decisions/`, `docs/04-plans/`, `docs/05-insights/`, and `docs/99-archive/`, with matching prefixed archive subfolders under `docs/99-archive/`.
- Keep durable governance policy in `README.md`, `AGENTS.md`, local ADRs, and the repository's folder structure.
- Route governed docs by intent: ADR = what we decided, plan = what we are doing now, research reduces uncertainty before a decision, ideas stay early and lightweight, and insights preserve execution or operational learning.
- Preserve local ADR numbering and intent. Existing ADRs `0001` through `0006` remain accepted local decisions of this repository after migration into `docs/03-decisions/`.
- Migrate pre-existing governed docs into canonical folders conservatively without renumbering them or silently rewriting their historical meaning.
- Treat historical governed docs created before [PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns](../04-plans/PLN-2026-04-21-01-modernize-governance-and-adopt-relevant-toolsmith-patterns.md) as pre-baseline records. They may retain legacy filenames or lighter metadata when conservative preservation is more accurate than retrofitting them.
- Require new live governed docs to follow the current metadata and naming rules described in `README.md` and `AGENTS.md`.
- Keep supporting operational guides outside the lifecycle under `docs/guides/` when they are not themselves ideas, research, decisions, plans, or insights.

## Consequences

The repository now has explicit local governance in canonical form while preserving its earlier ADR and planning history. Future governance audits should treat [PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns](../04-plans/PLN-2026-04-21-01-modernize-governance-and-adopt-relevant-toolsmith-patterns.md) as the baseline cutoff for historical-format drift in older plans, research, and similar closed records, but not for live governing docs.
