# Phase 3: Authority and Safety - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Enforce authoritative-layout rules on top of the Phase 2 resolver so `changes` fails loudly on ambiguity, never dual-writes during normal operation, and records schema/version compatibility as an explicit operational gate. This phase covers authority selection, authority-related failure classes, warning behavior for non-blocking sibling candidates, and write-entry-point enforcement; it does not add the richer `doctor` UX or migration prompt UX.

</domain>

<decisions>
## Implementation Decisions

### Authority failure model
- **D-01:** Ordinary ambiguity errors stay terse rather than embedding rich inline diagnostics
- **D-02:** User-facing authority failures remain distinct classes: `ambiguous`, `legacy_only`, `invalid_manifest`, and `uninitialized`
- **D-03:** Terse ambiguity errors must include the affected scope, a short failure statement, and the next `changes doctor` command to run

### Operational validity and mixed-candidate tolerance
- **D-04:** If exactly one candidate is operationally valid for a scope, ordinary commands may proceed even when non-operational sibling candidates also exist
- **D-05:** Non-operational sibling candidates (`invalid_manifest` or legacy-only) must trigger a warning rather than block normal operation when exactly one authoritative candidate exists
- **D-06:** Equivalent candidates that canonicalize to the same physical location collapse to one authority target rather than counting as ambiguity

### Warning behavior
- **D-07:** Non-blocking sibling-candidate warnings are surfaced by the CLI on stderr
- **D-08:** Lower layers should return structured warning information rather than user-facing warning strings
- **D-09:** The warning should appear for every ordinary command that resolves the affected scope, not only for write commands
- **D-10:** Phase 3 must not add suppression config for invalid or legacy sibling warnings

### Schema and compatibility
- **D-11:** `schema_version` compatibility is strict exact-version only for operational validity in Phase 3
- **D-12:** Candidates with unknown or unsupported schema versions are not operationally valid for ordinary commands

### Write safety enforcement
- **D-13:** No setup or write command, including `init`, gets a special authority exception in Phase 3
- **D-14:** The no-dual-write rule is enforced at each write entry point rather than through one centralized write API
- **D-15:** Write-entry-point enforcement should rely on shared authority-check helpers in `internal/config`, but each write-capable operation must invoke the gate explicitly

### the agent's Discretion
- The exact internal type names for authority warnings, authority check results, and write-gate helpers
- The precise stderr wording of terse warnings and ambiguity failures, as long as the approved minimum content is preserved
- Whether warning metadata is returned as dedicated structs, slices on existing results, or another structured form that cleanly separates policy from presentation

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Approved authority and layout rules
- `.planning/proposals/layout-resolution.md` — Governing proposal for authority rules, ambiguity behavior, legacy detection, warning boundaries, schema-version expectations, and no-dual-write requirements
- `.planning/REQUIREMENTS.md` — Phase-mapped requirements, especially `AUTH-01`, `AUTH-02`, and `MIGR-04`
- `.planning/ROADMAP.md` — Phase boundary, success criteria, and the split between Phase 3 authority enforcement and later Phase 4 `doctor` UX

### Locked prior-phase decisions
- `.planning/phases/01-layout-proposal/01-CONTEXT.md` — Locked policy decisions for ambiguity, doctor ownership, manifest validity, and migration boundaries
- `.planning/phases/02-resolution-core/02-CONTEXT.md` — Locked resolver architecture, candidate evidence model, path normalization rules, and compatibility-wrapper strategy
- `.planning/phases/02-resolution-core/02-VERIFICATION.md` — Verified behavior of the resolver core that Phase 3 must build on instead of replacing

### Existing layout baseline and implementation seams
- `docs/decisions/ADR-0001-repo-local-xdg-layout.md` — Baseline repo-local XDG rationale that Phase 3 must preserve while adding authority enforcement
- `internal/config/resolution_types.go` — Current scope/style/status/candidate contracts used by the resolver
- `internal/config/resolution.go` — Current candidate inspection, scope summarization, and preference behavior that Phase 3 authority rules will refine
- `internal/config/manifest.go` — Current manifest parsing, schema version reading, canonical comparison, and containment behavior
- `internal/config/config.go` — Current compatibility helper and config-load entry points that already surface some resolver status failures
- `internal/app/init.go` — Existing write/setup entry point that must obey the same authority rules as other commands
- `internal/cli/app.go` — Existing stdout/stderr and user-facing error surface for terse warnings and failures

### Codebase conventions and testing expectations
- `.planning/codebase/CONVENTIONS.md` — Existing Go error-handling and CLI stderr/stdout conventions that Phase 3 should follow
- `.planning/codebase/TESTING.md` — Existing Go test expectations and command baselines for verification

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/resolution_types.go`: existing `ResolutionStatus`, `Candidate`, `ScopeResolution`, and `LayoutManifest` types already capture most of the evidence Phase 3 needs
- `internal/config/resolution.go`: existing `summarizeScopeStatus`, `Preferred`, and `Authoritative` behavior give Phase 3 a direct place to refine ambiguity and mixed-candidate handling
- `internal/config/manifest.go`: existing canonical comparison and schema-version parsing already support equivalent-candidate collapse and strict-version gating
- `internal/config/config.go`: existing config-load and compatibility helper flow is already a real ordinary-operation entry point for authority failures
- `internal/app/init.go`: existing bootstrap and manifest-writing flow is the main write/setup path that must now obey the authority rules without a special exception
- `internal/cli/app.go`: existing stderr fail path and stdout success path are the natural presentation layer for terse ambiguity failures and warnings

### Established Patterns
- Path and config semantics remain centralized in `internal/config/` rather than being duplicated in app or CLI code
- CLI presentation belongs in `internal/cli/`, with lower layers returning structured data or wrapped errors
- Fail-fast wrapped errors are the established norm, and warnings should stay concise rather than verbose
- Compatibility wrappers are acceptable when they delegate to the authoritative core instead of preserving separate policy logic

### Integration Points
- Ordinary command flows already reach authority-sensitive state through `config.Load`, path helpers, and `Initialize`
- Phase 3 can introduce shared authority-check helpers inside `internal/config` and require each write-capable operation to invoke them before writing
- Phase 4 `doctor` work will consume the same candidate evidence and authority outcomes, so Phase 3 should avoid baking detailed human diagnostics into normal command errors

</code_context>

<specifics>
## Specific Ideas

- One valid authoritative candidate plus non-operational siblings should continue, but cleanup should stay visible through warnings and `doctor`
- Warnings should remain short enough for everyday CLI use, but consistent across ordinary commands that resolve the affected scope
- Equivalent candidates that collapse to the same physical location should not create false ambiguity

</specifics>

<deferred>
## Deferred Ideas

- Warning suppression config for specific invalid or legacy sibling paths — explicitly rejected for Phase 3 and deferred unless later experience proves it necessary
- Richer `doctor` explanation output and migration-oriented suggestion text — belongs to Phase 4, not Phase 3

</deferred>

---
*Phase: 03-authority-and-safety*
*Context gathered: 2026-04-06*
