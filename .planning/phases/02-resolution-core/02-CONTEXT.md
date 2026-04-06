# Phase 2: Resolution Core - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Implement the shared path-resolution layer for `changes` so both global and repo-local state can resolve through the approved `xdg` and `home` layouts. This phase is about the core resolution model, manifest interpretation, candidate evidence, and transition-friendly config APIs; it does not add the full `doctor` command UX or complete consumer rewiring across the whole application.

</domain>

<decisions>
## Implementation Decisions

### Resolver API shape
- **D-01:** The primary Phase 2 API should use explicit layout objects rather than helper-style path semantics
- **D-02:** Phase 2 should expose both `ResolveAll` and per-scope convenience entry points such as `ResolveGlobal` and `ResolveRepo`
- **D-03:** `ResolveAll` is the primary orchestration shape; per-scope entry points are thin convenience wrappers over the same core engine

### Manifest semantics
- **D-04:** Phase 2 fully owns manifest parsing, manifest validation, and symbolic-to-resolved path expansion in the core
- **D-05:** The core must preserve both symbolic manifest values and resolved filesystem paths so later diagnostics can explain what the operator wrote versus what the resolver used
- **D-06:** Manifest semantics in Phase 2 must conform to the approved Phase 1 `[layout]` schema and the allowed symbolic references without adding historical or churn-heavy fields

### Candidate evidence model
- **D-07:** Phase 2 should return evidence-rich candidate records for all detected layouts, not just the winning authoritative candidate
- **D-08:** The core returns structured statuses instead of forcing fatal errors at the resolver boundary; callers decide whether a given status is acceptable for their use case
- **D-09:** The candidate evidence model must be rich enough that later `doctor`, ambiguity, and migration phases can build on it without reconstructing filesystem evidence from scratch

### Path normalization and safety
- **D-10:** Phase 2 uses strict path normalization before candidate comparison and path resolution
- **D-11:** Strict normalization includes cleaning relative path segments, canonicalizing equivalent roots for comparison, and preventing repo-local resolved paths from escaping the repo root
- **D-12:** Normalization should make equivalent candidates compare as equivalent while still preserving original symbolic inputs for diagnostics

### Consumer transition strategy
- **D-13:** Existing config/path consumers may use compatibility wrappers during the transition instead of a hard cutover in Phase 2
- **D-14:** Compatibility wrappers should delegate to the new resolver core rather than preserving separate path logic
- **D-15:** The Phase 2 implementation should prove the core through real config package APIs, but avoid unnecessary churn in unrelated packages while the resolver stabilizes

### the agent's Discretion
- The exact internal type names for resolver objects, candidate records, and normalization helpers
- Whether the shared internal engine is split across multiple files inside `internal/config/` or kept together while the resolver model settles
- The exact balance between exported compatibility wrappers and unexported adapter helpers, as long as the explicit resolver object model stays primary

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Approved design and scope
- `.planning/proposals/layout-resolution.md` — Governing design artifact for supported styles, precedence, manifests, authority rules, config shape, and migration safety
- `.planning/PROJECT.md` — Project-level constraints, non-goals, and locked decisions for the layout initiative
- `.planning/REQUIREMENTS.md` — Phase-mapped requirements, especially `GLBL-01`, `GLBL-02`, `REPO-01`, `REPO-03`, and `MIGR-01`
- `.planning/ROADMAP.md` — Phase boundary, success criteria, and execution order
- `.planning/phases/01-layout-proposal/01-CONTEXT.md` — Phase 1 decisions that Phase 2 must treat as locked
- `.planning/phases/01-layout-proposal/01-implementation-gate.md` — Exact Phase 2 entry checklist derived from the approved proposal bundle

### Existing layout baseline
- `docs/decisions/ADR-0001-repo-local-xdg-layout.md` — Current repo-local XDG rationale and baseline assumptions the new resolver must preserve or intentionally evolve

### Current code integration points
- `internal/config/config.go` — Current centralized path/config layer that Phase 2 will replace or extend
- `internal/app/init.go` — Current init transaction flow, directory creation, and `.gitignore` integration
- `internal/app/app.go` — Current service-layer consumers of config/path helpers
- `internal/cli/app.go` — Existing CLI routing and command integration points for future `init global` and `doctor` work

### Codebase conventions and structure
- `.planning/codebase/CONVENTIONS.md` — Current Go package, error, and CLI patterns to preserve
- `.planning/codebase/STRUCTURE.md` — Current package boundaries and path-consumer layout
- `.planning/codebase/STACK.md` — Tooling and environment baseline for the Go CLI

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/config.go`: existing `Config` model, repo config loading, default path settings, and current helper surface such as `RepoConfigPath`, `FragmentsDir`, `ReleasesDir`, and `StateDir`
- `internal/app/init.go`: initialization transaction scaffolding, bootstrap artifact checks, and `.gitignore` handling that later phases will need to rewire onto the new resolver
- `internal/app/app.go`: central service workflows that currently depend on `config.Load` and path helpers through downstream packages
- `internal/cli/app.go`: current command parsing and repo-root flow that future Phase 4 command work will plug into

### Established Patterns
- Path and config semantics belong in `internal/config/`, not scattered across app or CLI packages
- Application services should consume structured results from `internal/config/` rather than duplicating path logic
- Fail-fast validation with explicit wrapped errors is the existing norm and should continue in the resolver core
- Compatibility transitions are acceptable when they centralize behavior instead of preserving duplicate logic

### Integration Points
- All current path helper callers across fragments, releases, render, changelog, and init ultimately depend on `internal/config/config.go`
- Phase 2 can establish the new resolver core inside `internal/config/` while leaving higher-level CLI/service policy decisions for later phases
- The resolver will eventually need to support both ordinary operational consumers and later diagnostic consumers, so candidate evidence must survive beyond a single happy-path return value

</code_context>

<specifics>
## Specific Ideas

- Keep the explicit layout-object model as the architectural truth even if compatibility wrappers remain temporarily exported
- Preserve original symbolic manifest values alongside resolved paths so later `doctor` output can explain both
- Treat strict normalization as part of safety, not just convenience, especially for repo-local path containment

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---
*Phase: 02-resolution-core*
*Context gathered: 2026-04-06*
