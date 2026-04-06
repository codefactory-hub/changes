# changes

## What This Is

`changes` is a Go CLI for managing release fragments, release records, and rendered release notes from canonical project history. The next project focus is to make its global and repo-local storage layout flexible so operators can use either XDG-style directories or a single-root `changes_home` layout without losing migration safety or clarity about which location is authoritative.

## Core Value

`changes` must make release metadata location predictable, inspectable, and safe even when multiple supported storage layouts are possible.

## Requirements

### Validated

- ✓ Users can initialize a repository for `changes` and create repo-local config, release-history prompts, and changelog scaffolding from the CLI — existing
- ✓ Users can create durable change fragments with semantic metadata and optional interactive authoring — existing
- ✓ Users can inspect current release state, recommended version bump, and prerelease lineage from file-backed state — existing
- ✓ Users can record release records and render release output through named profiles and templates — existing
- ✓ Users can keep release metadata in repo-local XDG-style paths resolved through a single config/path layer — existing

### Active

- [ ] Support a global single-root `CHANGES_HOME` layout alongside XDG-style global paths, with clear precedence and deterministic path resolution
- [ ] Support a repo-local single-root layout such as `./.changes` alongside repo-local XDG-style paths, with clear authoritative selection rules
- [ ] Fail safely when multiple valid layout roots exist and explain how the user should choose a single authoritative location
- [ ] Generate an LLM-oriented migration prompt that includes deterministically gathered origin and destination layout details
- [ ] Record low-churn structural layout metadata so `changes` can reason about supported layouts without rewriting manifests during ordinary commands
- [ ] Define and document clean command shapes for `init`, `init global`, and `doctor` flows that cover inspection, migration help, and initialization
- [ ] Keep write behavior single-target only; never dual-write to competing layouts
- [ ] Make the default behavior XDG-style while still allowing environment- or repo-level single-root overrides

### Out of Scope

- Automatic dual-write synchronization between layouts — this risks silent divergence and makes authoritative state ambiguous
- Silent conflict resolution when multiple supported layout roots already exist — the tool should stop and force an explicit choice
- Automatic destructive migration or auto-merge of competing layout roots — migration help should be explicit and reviewable first
- Unrelated release-model, render-profile, or changelog-format work — this effort is limited to storage layout resolution and migration UX

## Context

This is a brownfield CLI with a current repo-local XDG-style layout anchored in `internal/config/config.go` and used across `internal/app/`, `internal/fragments/`, `internal/releases/`, and `internal/render/`. The new effort is not about adding another ad hoc path helper; it is about designing a durable layout-resolution model that can support both global and repo-local storage styles, make precedence legible, preserve migration-safe reads, and expose command UX that users can understand before any file movement occurs.

The user wants the defaults and precedence model to be explicit, especially the distinction between global and per-repository layout choices. They also want proposal-quality command shapes and migration UX reviewed before implementation starts, then locked into planning artifacts before implementation begins. Existing codebase mapping in `.planning/codebase/` should be treated as reference material for where the current path assumptions live.

## Constraints

- **Compatibility**: Existing XDG-style repositories must continue to work — layout flexibility cannot strand current users
- **Safety**: Single authoritative write target only — no dual writes, silent merges, or hidden precedence
- **Clarity**: Global and repo-local layout behavior must be documented and inspectable — operators need to understand what path won and why
- **Migration**: Layout changes must preserve operator trust — migration assistance should be explicit, reproducible, and non-destructive by default
- **Scope**: Design and command UX must be agreed before implementation — proposal work comes first

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
*Last updated: 2026-04-06 after design lock-in*
