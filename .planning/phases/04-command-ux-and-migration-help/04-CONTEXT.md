# Phase 4: Command UX and Migration Help - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Expose the approved layout model through real operator-facing command UX and documentation. This phase covers the `doctor` inspection and migration-prompt surface, the user-facing layout-selection flags and success output for `init` / `init global`, and the documentation/examples that explain defaults, precedence, and repo-vs-global behavior. It does not introduce new layout semantics or change the authority model from earlier phases.

</domain>

<decisions>
## Implementation Decisions

### `doctor` command surface
- **D-01:** `changes doctor` remains a single flag-driven command family rather than gaining nested subcommands
- **D-02:** When running inside a repository, `changes doctor` defaults to repo scope
- **D-03:** `changes doctor` must still allow explicit `--scope global|repo|all`
- **D-04:** Migration help stays on the same command family as `changes doctor --migration-prompt`, not a separate top-level command

### Migration prompt output behavior
- **D-05:** `changes doctor --migration-prompt` prints the generated Markdown brief to stdout by default
- **D-06:** The migration-prompt command supports optional `--output PATH`
- **D-07:** When `--output PATH` is used, the prompt body goes to the file and stdout may print only a concise success line

### `init` and `init global` layout-selection UX
- **D-08:** `changes init` and `changes init global` keep the minimal approved flags: `--layout xdg|home` and `--home PATH`
- **D-09:** `--home PATH` is only valid when `--layout home`; for `home` repo init, omission still defaults to `.changes`
- **D-10:** Successful init output should explicitly state the selected layout and the resolved config, data, and state locations
- **D-11:** Init output should mention `.gitignore` updates only when the command actually changed the ignore file

### Documentation and examples
- **D-12:** Operator-facing docs should lead with the default path first, then explain alternatives, not start with a comparison matrix
- **D-13:** Documentation should explain global and repo-local behavior in that order: default behavior, then override behavior, then migration/inspection workflows
- **D-14:** The canonical examples for this phase are:
  - default repo init
  - repo init with `home`
  - global init with `home`
  - `doctor --scope repo --explain`
  - `doctor --scope global`
  - `doctor --migration-prompt --scope repo --to home`
- **D-15:** Documentation should include a concise precedence table for global bootstrap and repo initialization, but the table should support the narrative rather than replace it

### the agent's Discretion
- The exact help-text wording for `doctor`, `init`, and `init global`, as long as it reflects the locked command surface and precedence model
- Whether the canonical examples live primarily in `README.md`, command help text, or supporting docs, as long as the default-first documentation shape is preserved
- The exact formatting of the migration-prompt success line when `--output PATH` is used

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Approved design and requirements
- `.planning/proposals/layout-resolution.md` — Governing design artifact for command surface, precedence, manifest behavior, repo-state ignore rules, and migration-brief expectations
- `.planning/REQUIREMENTS.md` — Phase-mapped requirements, especially `GLBL-03`, `REPO-02`, `AUTH-03`, `MIGR-02`, `MIGR-03`, and `CMD-06`
- `.planning/ROADMAP.md` — Phase 4 boundary, success criteria, and relationship to Phase 5 rollout work
- `.planning/PROJECT.md` — Project-level non-goals and the overall purpose of flexible layout resolution

### Locked prior-phase decisions
- `.planning/phases/01-layout-proposal/01-CONTEXT.md` — Phase 1 decisions for `doctor`, `init`, `init global`, precedence, repo-state ignore rules, and migration-brief behavior
- `.planning/phases/02-resolution-core/02-CONTEXT.md` — Resolver architecture and compatibility-wrapper rules that Phase 4 command UX must build on
- `.planning/phases/03-authority-and-safety/03-CONTEXT.md` — Warning behavior, terse doctor-guided failures, and write-entry-point rules that Phase 4 must preserve
- `.planning/phases/03-authority-and-safety/03-VERIFICATION.md` — Verified authority behavior that the new command UX and docs must accurately represent

### Existing code and baseline docs
- `docs/decisions/ADR-0001-repo-local-xdg-layout.md` — Current repo-local XDG baseline and rationale that docs must evolve from without contradicting
- `internal/cli/app.go` — Existing top-level command routing, help style, stdout/stderr conventions, and warning/error presentation seam
- `internal/cli/app_integration_test.go` — Existing CLI integration-test home for command-output behavior and stderr/stdout expectations
- `internal/app/init.go` — Existing init orchestration, `.gitignore` updates, global `[repo.init]` defaults handling, and success-output data source
- `internal/config/config.go` — Existing config defaults, global/repo config paths, and compatibility helper surface that docs and UX must describe accurately
- `README.md` — Primary operator-facing documentation that will likely carry the default-first explanation and canonical examples

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/cli/app.go`: already owns top-level command routing and user-facing stdout/stderr formatting, making it the natural home for `doctor` flags and init success messaging
- `internal/cli/app_integration_test.go`: already covers CLI UX and is the right place for stderr/stdout and help-surface verification
- `internal/app/init.go`: already computes authoritative repo layout selection, global `[repo.init]` defaults, and `.gitignore` state ignore entries, so Phase 4 mostly needs to expose that behavior clearly
- `internal/config/config.go`: already contains default config/data/state paths and the compatibility semantics that user-facing docs must explain

### Established Patterns
- The CLI uses top-level commands with flag-driven sub-behavior rather than nested command trees
- Human-facing diagnostics belong in `internal/cli`, while lower layers return structured data
- Command help and README-level docs are the main operator-facing explanation surfaces in this repo
- Integration tests are the normal way to lock command wording and stdout/stderr behavior before broader rollout work

### Integration Points
- `doctor` command UX will plug into `internal/cli/app.go` and new app/service entry points backed by the resolver and authority model from Phases 2 and 3
- Init layout flags and success output must connect `internal/cli/app.go` to the existing selection data already produced in `internal/app/init.go`
- README/help updates must reflect the same precedence and path details emitted by the implementation so docs do not drift from real behavior

</code_context>

<specifics>
## Specific Ideas

- Start docs with the common path: “what happens by default if you just run `changes init`”
- Treat `home` as the explicit alternative rather than the first mental model
- Use the migration-prompt examples as operator-oriented workflows, not as low-level design exposition
- Keep the initial doc structure pragmatic and revisit deeper restructuring later only if usage shows that the comparison-first approach would be clearer

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---
*Phase: 04-command-ux-and-migration-help*
*Context gathered: 2026-04-07*
