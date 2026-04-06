# Roadmap: changes

## Overview

This roadmap starts with explicit design work so layout semantics, precedence, manifests, and migration UX are reviewable before implementation begins. After the proposal is agreed, the work moves through a single path-resolution core, ambiguity handling and schema metadata, command/documentation integration, and final rollout verification across existing XDG-style behavior and new single-root layouts.

## Phases

- [ ] **Phase 1: Layout Proposal** - Define the command shapes, precedence rules, authoritative-selection model, and migration UX before any implementation
- [ ] **Phase 2: Resolution Core** - Build the path model that can resolve XDG-style and single-root layouts for both global and repo-local state
- [ ] **Phase 3: Authority and Safety** - Add ambiguity detection, schema metadata, and fail-loud authority selection rules
- [ ] **Phase 4: Command UX and Migration Help** - Surface inspection and migration workflows through clean CLI commands and documentation
- [ ] **Phase 5: Rollout Verification** - Update init/default behavior and prove the full behavior through tests and migration-oriented scenarios

## Phase Details

### Phase 1: Layout Proposal
**Goal**: Produce and lock a concrete design proposal covering command shape, precedence, authoritative selection, manifests, and migration behavior before coding
**Depends on**: Nothing (first phase)
**Requirements**: [CMD-01, CMD-02, CMD-03, CMD-04, CMD-05]
**Success Criteria** (what must be TRUE):
  1. A written proposal explains global vs repo-local layout choices and their defaults
  2. The precedence model is explicit enough that a user can predict which layout wins
  3. Final command shapes for `init`, `init global`, and `doctor` are documented and locked before implementation
**Plans**: 2 plans

Plans:
- [ ] 01-01: Draft the layout-resolution and precedence proposal
- [ ] 01-02: Review proposal tradeoffs and refine command UX before implementation

### Phase 2: Resolution Core
**Goal**: Implement the core path-resolution layer for XDG-style and single-root layouts without changing write safety guarantees
**Depends on**: Phase 1
**Requirements**: [GLBL-01, GLBL-02, REPO-01, REPO-03, MIGR-01]
**Success Criteria** (what must be TRUE):
  1. `changes` can resolve global config, data, and state paths from either supported layout style
  2. `changes` can resolve repo-local config, data, and state paths from either supported layout style
  3. The active layout can be determined through one shared core model rather than scattered path heuristics
**Plans**: 2 plans

Plans:
- [ ] 02-01: Design and implement the shared layout model and resolver APIs
- [ ] 02-02: Wire global and repo-local path consumers to the new resolver core

### Phase 3: Authority and Safety
**Goal**: Enforce authoritative-layout rules and make ambiguity visible, inspectable, and non-destructive
**Depends on**: Phase 2
**Requirements**: [AUTH-01, AUTH-02, MIGR-04]
**Success Criteria** (what must be TRUE):
  1. Commands fail loudly when multiple supported layouts compete for authority
  2. Errors explain the competing candidates and suggest how the operator can choose one
  3. Normal command execution still writes to exactly one authoritative target
**Plans**: 2 plans

Plans:
- [ ] 03-01: Add authoritative-layout detection and competing-layout diagnostics
- [ ] 03-02: Add schema/version metadata and enforce single-target write behavior

### Phase 4: Command UX and Migration Help
**Goal**: Expose the new behavior through clear commands, migration prompt generation, and updated docs
**Depends on**: Phase 3
**Requirements**: [GLBL-03, REPO-02, AUTH-03, MIGR-02, MIGR-03, CMD-06]
**Success Criteria** (what must be TRUE):
  1. Users can inspect active layout resolution and understand why it was chosen
  2. Users can generate migration help that includes deterministic source/destination facts
  3. Docs explain the difference between global overrides and repo-local overrides with concrete examples, and repo-local init ignores the active `state` directory consistently
**Plans**: 2 plans

Plans:
- [ ] 04-01: Implement CLI command surfaces for layout inspection and migration prompt generation
- [ ] 04-02: Document the commands, defaults, precedence, and merge/migration guidance

### Phase 5: Rollout Verification
**Goal**: Make the new layout model the default-safe behavior and verify compatibility across existing and new layouts
**Depends on**: Phase 4
**Requirements**: [none]
**Success Criteria** (what must be TRUE):
  1. `changes init` defaults are aligned with the approved layout model
  2. Existing XDG-style repositories still behave correctly without manual migration
  3. Automated tests cover resolution precedence, ambiguity failures, and migration-prompt generation
**Plans**: 2 plans

Plans:
- [ ] 05-01: Align init/default selection behavior with the approved design
- [ ] 05-02: Add regression, precedence, and migration-oriented verification coverage

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Layout Proposal | 0/2 | Not started | - |
| 2. Resolution Core | 0/2 | Not started | - |
| 3. Authority and Safety | 0/2 | Not started | - |
| 4. Command UX and Migration Help | 0/2 | Not started | - |
| 5. Rollout Verification | 0/2 | Not started | - |
