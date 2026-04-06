# Phase 1: Layout Proposal - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Define and lock the layout-resolution design for `changes` before implementation begins. This phase covers the proposal-quality decisions for supported layout styles, precedence, authoritative layout selection, manifest behavior, migration prompt behavior, and command UX, but does not implement the resolver itself.

</domain>

<decisions>
## Implementation Decisions

### Supported layout model
- **D-01:** The only supported layout styles are `xdg` and `home`
- **D-02:** Layout resolution happens independently for `global` and `repo` scopes
- **D-03:** Built-in behavior may choose default locations, but it must not introduce a third layout style

### Precedence and bootstrap
- **D-04:** Global bootstrap precedence is: explicit command flags, `CHANGES_HOME`, XDG env vars, then built-in default locations
- **D-05:** Repo init precedence is: explicit command flags, global config repo-init defaults, `CHANGES_HOME` as a style preference signal, XDG env vars as a style preference signal, then built-in default locations
- **D-06:** `CHANGES_HOME` beats XDG env as a preference signal, but it must not silently override a conflicting on-disk ambiguity

### Manifest schema and validity
- **D-07:** Layout manifests are structural, symbolic, and low-churn; ordinary commands must not rewrite them
- **D-08:** Manifest fields are expressed under `[layout]`, using symbolic references such as `$REPO_ROOT`, `$CHANGES_HOME`, and `$layout.root`
- **D-09:** Manifest-backed candidates are operationally valid only when `layout.toml` parses and matches the candidate `scope` and `style`
- **D-10:** Legacy layouts without a manifest are detectable, but not operationally valid
- **D-11:** A legacy candidate is considered real enough to diagnose if it matches a supported layout shape and contains an authoritative `changes` artifact; `config.toml` alone is enough

### Authority and repair
- **D-12:** Multiple supported candidates for the same scope are always an ambiguity error
- **D-13:** Ordinary commands must fail when they only find legacy-detected layouts; `changes doctor` is the only command that may inspect those situations
- **D-14:** Manifest stamping or repair is explicit, not opportunistic

### Command UX
- **D-15:** Initialization stays under `changes init` and `changes init global`
- **D-16:** Diagnosis, ambiguity inspection, and migration assistance stay under `changes doctor`
- **D-17:** `changes doctor` uses tiered output: concise default output, richer `--explain` output, and structured `--json` output
- **D-18:** Default `doctor` output should stay quiet; richer candidate analysis belongs behind `--explain`

### Migration prompt behavior
- **D-19:** Migration help is generated as an LLM-oriented structured brief, not as executable shell or file operations
- **D-20:** The generated prompt must include source and destination layout details, artifact inventories, ambiguity/conflict notes, and a required verification section
- **D-21:** The generated prompt must instruct the external tool to preserve exactly one authoritative destination and avoid dual-write outcomes

### Config and repo hygiene
- **D-22:** Global config may only express repo-init defaults under `[repo.init]`
- **D-23:** `[repo.init].home` is valid only when `style = "home"`; if omitted for `home`, it defaults to `.changes`
- **D-24:** Repo-local initialization must keep the authoritative repo-local `state` directory ignored consistently in `.gitignore`

### the agent's Discretion
- The exact internal type names, helper names, and package decomposition for the eventual resolver
- The precise JSON schema used by `changes doctor --json`, as long as it fully represents the approved diagnostic model
- The wording and layout of `doctor --explain` output, as long as it preserves the approved tiers and diagnostics

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Approved design
- `.planning/proposals/layout-resolution.md` — Governing design artifact for supported styles, precedence, manifests, command surface, and migration prompt behavior
- `.planning/PROJECT.md` — Project-level constraints, non-goals, and locked decisions for this initiative
- `.planning/REQUIREMENTS.md` — Phase-mapped requirements and command/documentation expectations
- `.planning/ROADMAP.md` — Phase boundary, success criteria, and execution order for this work

### Existing layout baseline
- `docs/decisions/ADR-0001-repo-local-xdg-layout.md` — Current repo-local XDG layout rationale and baseline assumptions that the new model must evolve from
- `internal/config/config.go` — Current XDG-style path defaults and central path-resolution layer that the new design will replace or extend

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/config.go`: Existing centralized path helper layer for config, data, state, prompts, templates, and changelog paths
- `internal/app/init.go`: Existing init flow, including directory creation, repo bootstrap behavior, and `.gitignore` updates
- `internal/cli/app.go`: Existing command parsing, help text, and stdout/stderr conventions for adding `init global` and `doctor`
- `internal/reporoot/reporoot.go`: Existing repo-root detection used by repo-scoped commands
- `internal/config/config_test.go`, `internal/app/app_test.go`, `internal/cli/app_integration_test.go`: Existing test homes for config semantics, init flows, and CLI integration behavior

### Established Patterns
- Centralize path and config semantics in `internal/config/` rather than scattering filesystem knowledge across packages
- Keep command routing and human-readable diagnostics in `internal/cli/`, with service orchestration in `internal/app/`
- Use fail-fast validation with explicit wrapped errors rather than silent fallback behavior
- Prefer docs-first decisions and explicit state transitions before altering repo or global layout behavior

### Integration Points
- Resolver decisions will affect every caller that currently uses `RepoConfigPath`, `FragmentsDir`, `ReleasesDir`, `TemplatesDir`, `PromptsDir`, and `StateDir` in `internal/config/config.go`
- `changes init` and future `changes init global` behavior will need to plug into `internal/app/init.go`
- `changes doctor` will need a new CLI/service path while following the existing output conventions in `internal/cli/app.go`

</code_context>

<specifics>
## Specific Ideas

- Built-in precedence wording should use “built-in default locations,” not “built-in XDG defaults”
- Manifest paths should be hierarchical and symbolic under `[layout]`, with `$layout.root` used only when a root exists
- Global bootstrap logic must not duplicate or restate values in global config that are required to discover the global config itself
- `changes doctor` should expose explanation depth through `--explain`, not by making the default output noisy

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---
*Phase: 01-layout-proposal*
*Context gathered: 2026-04-06*
