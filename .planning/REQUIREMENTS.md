# Requirements: changes

**Defined:** 2026-04-06
**Core Value:** `changes` must make release metadata location predictable, inspectable, and safe even when multiple supported storage layouts are possible.

## v1 Requirements

### Global Layout Resolution

- [x] **GLBL-01**: `changes` prefers `CHANGES_HOME` over XDG environment variables when resolving the active global layout
- [x] **GLBL-02**: `changes` can resolve global config, data, and state paths from either XDG-style directories or a single-root global layout
- [ ] **GLBL-03**: `changes` can explain which global layout source won and why

### Repository Layout Resolution

- [x] **REPO-01**: `changes` can resolve repo-local config, data, and state paths from either the default repo-local XDG-style layout or a declared single-root repo-local layout
- [ ] **REPO-02**: `changes init` can choose the repo-local layout style using a clean, documented command shape
- [ ] **REPO-03**: Repo-local layout selection rules are deterministic when no existing repo-local layout artifacts are present

### Ambiguity and Authority

- [ ] **AUTH-01**: If multiple supported global or repo-local layouts exist at once, `changes` fails instead of choosing silently
- [ ] **AUTH-02**: Ambiguity errors identify the competing authoritative candidates and suggest how to choose between them
- [ ] **AUTH-03**: Ambiguity errors can direct the user to generate an LLM prompt to help merge or migrate to one authoritative location

### Migration and Schema Metadata

- [x] **MIGR-01**: `changes` records structural layout schema metadata in managed layouts without rewriting it during ordinary command execution
- [ ] **MIGR-02**: `changes` can generate an LLM prompt for migrating between supported layouts using deterministically gathered source and destination details
- [ ] **MIGR-03**: Migration assistance includes origin metadata, destination metadata, and any detected ambiguity or conflict signals
- [ ] **MIGR-04**: `changes` never dual-writes to origin and destination layouts during normal operation

### Commands and Documentation

- [x] **CMD-01**: `changes doctor` can inspect active layout resolution, precedence, and ambiguity state for global and repo scopes
- [x] **CMD-02**: `changes doctor --migration-prompt` can generate migration help between supported layouts
- [x] **CMD-03**: Documentation explains the defaults, precedence, and difference between global vs repo-local layout overrides
- [x] **CMD-04**: `changes init` and `changes init global` expose clean, documented layout-selection flags
- [x] **CMD-05**: Documentation includes proposal-quality examples before implementation details are finalized
- [ ] **CMD-06**: Repo-local initialization updates `.gitignore` so the authoritative repo-local `state` directory is ignored consistently

## v2 Requirements

### Extended Migration Tooling

- **MIGR-05**: `changes` can validate a user-completed migration result against the expected source and destination layouts
- **MIGR-06**: `changes` can support future directory schema revisions beyond the first flexible-layout rollout

## Out of Scope

| Feature | Reason |
|---------|--------|
| Dual-write replication between layouts | Creates divergence risk and defeats authoritative storage selection |
| Silent automatic merge of two existing layouts | Too risky for release metadata without explicit operator review |
| Automatic destructive migration | Proposal-first migration help is safer than rewriting operator data unprompted |
| Broad redesign of unrelated release/render features | This project is focused on storage layout resolution, migration UX, and path semantics |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| GLBL-01 | Phase 2 | Complete |
| GLBL-02 | Phase 2 | Complete |
| GLBL-03 | Phase 4 | Pending |
| REPO-01 | Phase 2 | Complete |
| REPO-02 | Phase 4 | Pending |
| REPO-03 | Phase 2 | Pending |
| AUTH-01 | Phase 3 | Pending |
| AUTH-02 | Phase 3 | Pending |
| AUTH-03 | Phase 4 | Pending |
| MIGR-01 | Phase 2 | Complete |
| MIGR-02 | Phase 4 | Pending |
| MIGR-03 | Phase 4 | Pending |
| MIGR-04 | Phase 3 | Pending |
| CMD-01 | Phase 1 | Complete |
| CMD-02 | Phase 1 | Complete |
| CMD-03 | Phase 1 | Complete |
| CMD-04 | Phase 1 | Complete |
| CMD-05 | Phase 1 | Complete |
| CMD-06 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-06*
*Last updated: 2026-04-06 after design lock-in*
