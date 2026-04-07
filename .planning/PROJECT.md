# changes

## What This Is

`changes` is a Go CLI for managing release fragments, release records, and rendered release notes from canonical project history. As of `0.1.0-rc.2`, it supports flexible global and repo-local storage layouts through `xdg` and `home` styles, manifest-backed authority selection, `doctor`-based inspection, migration-oriented guidance, and explicit repo-local repair for legacy no-manifest repositories.

## Core Value

`changes` must make release metadata location predictable, inspectable, and safe even when multiple supported storage layouts are possible.

## Current State

- Shipped milestone: `0.1.0-rc.2` on 2026-04-07
- Current operator model:
  - `xdg` remains the default layout style
  - `home` is supported for global `CHANGES_HOME` and repo-local `.changes` style layouts
  - ordinary commands require manifest-backed authoritative layouts
  - legacy no-manifest layouts are diagnosable with `changes doctor`
  - legacy repo-local layouts can be repaired with `changes doctor --scope repo --repair`
  - broader layout migrations still use `changes doctor --migration-prompt`
- Current milestone archive:
  - Roadmap: `.planning/milestones/v0.1.0-rc.2-ROADMAP.md`
  - Requirements: `.planning/milestones/v0.1.0-rc.2-REQUIREMENTS.md`
  - Roadmap: `.planning/milestones/v0.1.0-rc.1-ROADMAP.md`
  - Requirements: `.planning/milestones/v0.1.0-rc.1-REQUIREMENTS.md`

## Requirements

### Validated

- ✓ Users can initialize a repository for `changes` and create repo-local config, release-history prompts, and changelog scaffolding from the CLI — existing
- ✓ Users can create durable change fragments with semantic metadata and optional interactive authoring — existing
- ✓ Users can inspect current release state, recommended version bump, and prerelease lineage from file-backed state — existing
- ✓ Users can record release records and render release output through named profiles and templates — existing
- ✓ Users can keep release metadata in repo-local XDG-style paths resolved through a single config/path layer — existing
- ✓ Users can resolve global config, data, and state through either XDG-style directories or a single-root `CHANGES_HOME` layout — `0.1.0-rc.1`
- ✓ Users can resolve repo-local config, data, and state through either repo-local XDG-style directories or a single-root `home` layout — `0.1.0-rc.1`
- ✓ Users can inspect active layout authority, precedence, and ambiguity through `changes doctor` — `0.1.0-rc.1`
- ✓ Users can generate migration-oriented layout briefs with deterministic source and destination facts — `0.1.0-rc.1`
- ✓ Users get fail-loud single-target writes, explicit ambiguity handling, and rollout-safe init defaults for layout management — `0.1.0-rc.1`
- ✓ Users can repair a legacy repo-local layout with `changes doctor --scope repo --repair` instead of creating `layout.toml` by hand — `0.1.0-rc.2`
- ✓ Repo-local repair remains single-target, fail-loud on ambiguity, and preserves the authoritative state ignore rule — `0.1.0-rc.2`
- ✓ Docs now explain when repair is appropriate versus when migration prompts are still required — `0.1.0-rc.2`

### Active

- [ ] Validate operator-completed migrations against the expected source and destination layouts
- [ ] Support future directory schema revisions beyond the first flexible-layout rollout

## Next Milestone Goals

- Validate completed operator migrations against the expected source and destination layouts
- Define how future directory schema revisions should be introduced without weakening current authority guarantees
- Decide whether global legacy-layout repair automation is warranted after operator feedback on the repo-local repair flow

## Out of Scope

- Automatic dual-write synchronization between layouts — this risks silent divergence and makes authoritative state ambiguous
- Silent conflict resolution when multiple supported layout roots already exist — the tool should stop and force an explicit choice
- Automatic destructive migration or auto-merge of competing layout roots — migration help should be explicit and reviewable first
- Unrelated release-model, render-profile, or changelog-format work — this effort is limited to storage layout resolution and migration UX

## Context

This is now a brownfield CLI that has shipped the first flexible-layout milestone and its first repair follow-up. The codebase includes a shared resolver core in `internal/config/`, authority-aware app and CLI flows in `internal/app/` and `internal/cli/`, rollout coverage that locks the precedence and compatibility boundary, and an explicit repo-local repair workflow for legacy layouts. The current product state remains intentionally conservative: manifest-backed layouts are the operational boundary, repairs are explicit, and broader migrations still rely on `doctor` guidance rather than hidden compatibility heuristics.

The next work should build on that shipped boundary rather than reopening it casually. Existing codebase mapping in `.planning/codebase/` and the archived milestone docs should be treated as the reference record for how the current layout model was introduced.

## Constraints

- **Compatibility**: Manifest-backed repositories must continue to work without regression; legacy repositories without manifests must fail cleanly and remain diagnosable
- **Safety**: Single authoritative write target only — no dual writes, silent merges, or hidden precedence
- **Clarity**: Global and repo-local layout behavior must be documented and inspectable — operators need to understand what path won and why
- **Migration**: Layout changes must preserve operator trust — migration assistance should be explicit, reproducible, and non-destructive by default
- **Scope**: Future milestones should extend the shipped layout model rather than relitigating the Phase 1-5 contract without a new proposal

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep `xdg` and `home` as the only supported layout styles | The model should be flexible without introducing an open-ended layout taxonomy | ✓ Good |
| Keep XDG as the default layout style | It matches the current system and should remain the baseline behavior | ✓ Good |
| Support a single-root `CHANGES_HOME` for global state | Users want a simpler override that can wrap config, data, and state together | ✓ Good |
| Support a single-root repo-local layout such as `./.changes` | Repositories may want one authoritative tool root instead of split XDG-style folders | ✓ Good |
| Treat multiple discovered supported layouts as an error | The tool should force users to pick one authoritative source instead of guessing | ✓ Good |
| Provide migration help by generating an LLM prompt with gathered source/destination details | The user wants explicit merge/migration assistance without dual-write or blind automation | ✓ Good |
| Keep manifests structural, symbolic, and low-churn | Layout metadata should help detection and migration without noisy updates or sandbox churn | ✓ Good |
| Use `[repo.init]` in global config for repo-init defaults | Hierarchical config is clearer than mangled key names and avoids bootstrap confusion | ✓ Good |
| Use `init` and `doctor` as the primary command families | Initialization and diagnosis/migration should be clear without a separate vague `layout` namespace | ✓ Good |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 after `0.1.0-rc.2` milestone*
