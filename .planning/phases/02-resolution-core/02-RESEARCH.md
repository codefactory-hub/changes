# Phase 2: Resolution Core - Research

**Researched:** 2026-04-06 [VERIFIED: current date]
**Domain:** Go CLI path-resolution core, manifest parsing, and compatibility-layer design for dual layout support. [VERIFIED: .planning/ROADMAP.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]
**Confidence:** HIGH overall; architecture details that depend on internal type/file names are MEDIUM. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; ASSUMED]

<user_constraints>
## User Constraints (from CONTEXT.md)

Verbatim copy from `.planning/phases/02-resolution-core/02-CONTEXT.md`. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

### Locked Decisions
- **D-01:** The primary Phase 2 API should use explicit layout objects rather than helper-style path semantics
- **D-02:** Phase 2 should expose both `ResolveAll` and per-scope convenience entry points such as `ResolveGlobal` and `ResolveRepo`
- **D-03:** `ResolveAll` is the primary orchestration shape; per-scope entry points are thin convenience wrappers over the same core engine
- **D-04:** Phase 2 fully owns manifest parsing, manifest validation, and symbolic-to-resolved path expansion in the core
- **D-05:** The core must preserve both symbolic manifest values and resolved filesystem paths so later diagnostics can explain what the operator wrote versus what the resolver used
- **D-06:** Manifest semantics in Phase 2 must conform to the approved Phase 1 `[layout]` schema and the allowed symbolic references without adding historical or churn-heavy fields
- **D-07:** Phase 2 should return evidence-rich candidate records for all detected layouts, not just the winning authoritative candidate
- **D-08:** The core returns structured statuses instead of forcing fatal errors at the resolver boundary; callers decide whether a given status is acceptable for their use case
- **D-09:** The candidate evidence model must be rich enough that later `doctor`, ambiguity, and migration phases can build on it without reconstructing filesystem evidence from scratch
- **D-10:** Phase 2 uses strict path normalization before candidate comparison and path resolution
- **D-11:** Strict normalization includes cleaning relative path segments, canonicalizing equivalent roots for comparison, and preventing repo-local resolved paths from escaping the repo root
- **D-12:** Normalization should make equivalent candidates compare as equivalent while still preserving original symbolic inputs for diagnostics
- **D-13:** Existing config/path consumers may use compatibility wrappers during the transition instead of a hard cutover in Phase 2
- **D-14:** Compatibility wrappers should delegate to the new resolver core rather than preserving separate path logic
- **D-15:** The Phase 2 implementation should prove the core through real config package APIs, but avoid unnecessary churn in unrelated packages while the resolver stabilizes

### Claude's Discretion
- The exact internal type names for resolver objects, candidate records, and normalization helpers
- Whether the shared internal engine is split across multiple files inside `internal/config/` or kept together while the resolver model settles
- The exact balance between exported compatibility wrappers and unexported adapter helpers, as long as the explicit resolver object model stays primary

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

Requirement descriptions copied from `.planning/REQUIREMENTS.md`. [VERIFIED: .planning/REQUIREMENTS.md]

| ID | Description | Research Support |
|----|-------------|------------------|
| GLBL-01 | `changes` prefers `CHANGES_HOME` over XDG environment variables when resolving the active global layout | Implement a single global precedence table inside the resolver and expose the winning input as structured evidence, not scattered env checks. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] |
| GLBL-02 | `changes` can resolve global config, data, and state paths from either XDG-style directories or a single-root global layout | Model global layouts as typed objects with `Config`, `Data`, and `State` paths plus manifest-backed symbolic/resolved forms. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] |
| REPO-01 | `changes` can resolve repo-local config, data, and state paths from either the default repo-local XDG-style layout or a declared single-root repo-local layout | Keep repo resolution in `internal/config` and guard repo-local expansions with strict containment checks before producing resolved paths. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: docs/decisions/ADR-0001-repo-local-xdg-layout.md; CITED: https://pkg.go.dev/path/filepath#IsLocal] |
| REPO-03 | Repo-local layout selection rules are deterministic when no existing repo-local layout artifacts are present | Add an explicit default-selection helper for init/bootstrap instead of inferring defaults from helper call sites. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] |
| MIGR-01 | `changes` records structural layout schema metadata in managed layouts without rewriting it during ordinary command execution | Keep manifest encode/decode separate from ordinary resolve operations; ordinary loads should parse and report, not rewrite. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- None — the repo root does not contain `CLAUDE.md`. [VERIFIED: repo file check]

## Project Constraints (from AGENTS.md)

- Prefer plain `go test ./...` first in this repository. [VERIFIED: AGENTS.md]
- Only add writable-path overrides if the environment actually requires them, starting with `GOCACHE`. [VERIFIED: AGENTS.md]

## Summary

Phase 2 should keep `internal/config` as the single path and config seam, but replace the current helper-only model with an explicit resolver that returns typed layout objects, candidate evidence, manifest metadata, and per-scope statuses for both `global` and `repo`. [VERIFIED: internal/config/config.go; VERIFIED: internal/app/init.go; VERIFIED: internal/app/app.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

The current codebase centralizes path semantics in `internal/config`, and every durable state consumer flows through helpers like `RepoConfigPath`, `FragmentsDir`, `ReleasesDir`, `TemplatesDir`, and `StateDir`; that makes `internal/config` the correct place to land the resolver core and compatibility wrappers without churning unrelated packages in Phase 2. [VERIFIED: internal/config/config.go; VERIFIED: internal/app/init.go; VERIFIED: internal/fragments/fragments.go; VERIFIED: internal/releases/releases.go; VERIFIED: internal/render/render.go; VERIFIED: .planning/codebase/STRUCTURE.md]

The safest Go implementation is a two-layer design: a typed resolver engine that inspects supported candidates and preserves symbolic plus resolved manifest data, plus thin compatibility helpers that delegate to the engine so existing callers keep working while Phase 3 and Phase 4 add ambiguity UX and full command surfaces. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/ROADMAP.md; ASSUMED]

**Primary recommendation:** Implement a typed `ResolveAll` engine in `internal/config` that owns precedence, manifest parsing, candidate evidence, and path normalization, then re-base the existing exported helper functions on that engine instead of adding new ad hoc joins. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: internal/config/config.go; ASSUMED]

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library (`os`, `path/filepath`, `errors`, `flag`) | go1.26.1 installed / module targets go1.26.0. [VERIFIED: go version; VERIFIED: go.mod] | Environment lookup, path cleaning, lexical containment, symlink evaluation, wrapped errors, and CLI flag behavior. [CITED: https://pkg.go.dev/os#UserConfigDir; CITED: https://pkg.go.dev/os#UserHomeDir; CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; CITED: https://pkg.go.dev/path/filepath#Rel; CITED: https://pkg.go.dev/errors#Is; CITED: https://pkg.go.dev/flag#FlagSet] | The repo already uses stdlib-only CLI and path handling, and the standard library provides the exact primitives this phase needs without adding dependencies. [VERIFIED: internal/cli/app.go; VERIFIED: internal/config/config.go; VERIFIED: .planning/codebase/STACK.md] |
| `github.com/BurntSushi/toml` | v1.6.0, published 2025-12-18. [VERIFIED: go list -m -json all] | Strict decode and encode for `config.toml` and new `layout.toml` manifests. [CITED: https://context7.com/burntsushi/toml/llms.txt] | The repo already uses it as the canonical TOML library, and it supports `DecodeFile` plus `MetaData.Undecoded()` for strict manifest validation. [VERIFIED: go.mod; VERIFIED: internal/config/config.go; CITED: https://context7.com/burntsushi/toml/llms.txt] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go `testing` package | Same toolchain as module. [VERIFIED: go version; VERIFIED: go.mod] | Table-driven temp-dir tests for precedence, manifest validation, and compatibility wrappers. [VERIFIED: .planning/codebase/TESTING.md] | Use for all new resolver tests; the repo does not use `testify`, `go-cmp`, or parallel test patterns today. [VERIFIED: .planning/codebase/TESTING.md] |
| Local `internal/reporoot` package | Repo-local package, no external version. [VERIFIED: .planning/codebase/STRUCTURE.md] | Existing repo-root discovery for commands that need a repo scope. [VERIFIED: .planning/codebase/STRUCTURE.md] | Reuse at CLI/service boundaries; do not duplicate repo-root detection inside the resolver. [VERIFIED: internal/cli/app.go; VERIFIED: internal/reporoot/reporoot.go; ASSUMED] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Stdlib `flag` plus current CLI style | Cobra/Viper | Not justified in Phase 2; the repo standard is stdlib `flag`, and this phase is core path architecture rather than CLI surface expansion. [VERIFIED: internal/cli/app.go; VERIFIED: .planning/codebase/STACK.md] |
| `BurntSushi/toml` strict decode | Custom manifest parser | Would duplicate an existing dependency and lose `Undecoded()`-based schema enforcement already used in `config.Load`. [VERIFIED: internal/config/config.go; CITED: https://context7.com/burntsushi/toml/llms.txt] |
| Compatibility wrappers in `internal/config` | Immediate whole-repo cutover to typed resolver consumers | The context explicitly allows wrappers in Phase 2, and an immediate cutover would add churn before resolver semantics stabilize. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md] |

**Installation:** No new dependencies are required for Phase 2; the needed Go modules are already in `go.mod`. [VERIFIED: go.mod]

```bash
go mod download
```

**Version verification:** For this Go phase, use `go list -m -json all` rather than `npm view`; it verified `github.com/BurntSushi/toml` `v1.6.0` and `github.com/Masterminds/semver/v3` `v3.4.0` in the current module graph. [VERIFIED: go list -m -json all]

## Architecture Patterns

### Recommended Project Structure

```text
internal/config/
├── resolution.go         # ResolveAll / ResolveGlobal / ResolveRepo orchestration [ASSUMED]
├── resolution_types.go   # scope, style, status, manifest, candidate, evidence types [ASSUMED]
├── manifest.go           # layout.toml decode, validate, symbolic expansion helpers [ASSUMED]
├── compat_paths.go       # RepoConfigPath/FragmentsDir/... wrappers backed by resolver [ASSUMED]
└── resolution_test.go    # precedence, normalization, manifest, and wrapper tests [ASSUMED]
```

Keeping everything in `internal/config/` matches the current package boundary and the codebase guidance that path and config semantics belong there rather than in app or CLI packages. [VERIFIED: .planning/codebase/STRUCTURE.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

### Pattern 1: Explicit Layout Object Model

**What:** Represent each supported layout as typed data for `scope`, `style`, `status`, resolved paths, symbolic manifest values, and evidence instead of returning only pre-joined strings. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/proposals/layout-resolution.md]

**When to use:** Use this model as the only core API; exported helper functions should read from it instead of re-implementing path semantics. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

**Recommended shape:** A `ResolveAll` result should contain one `ScopeResolution` for `global` and one for `repo`, each with `Status`, `Authoritative`, and `Candidates` fields, while `ResolveGlobal` and `ResolveRepo` remain thin wrappers. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; ASSUMED]

**Example:** [ASSUMED]

```go
type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeRepo   Scope = "repo"
)

type Style string

const (
	StyleXDG  Style = "xdg"
	StyleHome Style = "home"
)

type ResolutionStatus string

const (
	StatusUninitialized ResolutionStatus = "uninitialized"
	StatusResolved      ResolutionStatus = "resolved"
	StatusLegacyOnly    ResolutionStatus = "legacy_only"
	StatusAmbiguous     ResolutionStatus = "ambiguous"
	StatusInvalid       ResolutionStatus = "invalid"
)

type LayoutPaths struct {
	Root   string
	Config string
	Data   string
	State  string
}

type LayoutManifest struct {
	SchemaVersion int
	Scope         Scope
	Style         Style
	Symbolic      LayoutPaths
	Resolved      LayoutPaths
}

type Candidate struct {
	Scope    Scope
	Style    Style
	Status   ResolutionStatus
	Paths    LayoutPaths
	Manifest *LayoutManifest
	Evidence []Evidence
}
```

### Pattern 2: Two-Stage Path Normalization

**What:** Normalize candidate anchors first, then derive child paths lexically from those anchors; use `filepath.Clean`, `filepath.Join`, and `filepath.IsLocal` for lexical safety, and use `filepath.EvalSymlinks` only where an existing anchor must be canonicalized for equivalence checks. [CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#Join; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED]

**When to use:** Apply this to repo-local `home` roots such as `./.changes`, repo-local XDG subpaths, and manifest expansion of `$layout.root/...` values. [VERIFIED: .planning/proposals/layout-resolution.md]

**Why it matters:** `filepath.IsLocal` guarantees lexical containment but explicitly does not account for symlinks, so equivalence comparisons across existing roots need a canonical-anchor step if the repo root or env-derived root may be symlinked. [CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED]

**Example:** [CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#Join]

```go
func joinRepoLocal(base, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("resolver: %q escapes repo root", rel)
	}
	return filepath.Join(base, clean), nil
}
```

### Pattern 3: Compatibility Wrappers at the Package Boundary

**What:** Keep functions like `RepoConfigPath`, `FragmentsDir`, `ReleasesDir`, `TemplatesDir`, and `StateDir`, but implement them by resolving the repo scope once and reading the authoritative typed result. [VERIFIED: internal/config/config.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

**When to use:** Use wrappers during Phase 2 and Phase 3 while app, fragment, release, and render packages are still wired to the old helper surface. [VERIFIED: .planning/ROADMAP.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

**Why it fits this repo:** `internal/app/init.go`, `internal/fragments/fragments.go`, `internal/releases/releases.go`, and `internal/render/render.go` already depend on those helpers, so swapping the implementation underneath them is the lowest-churn migration path. [VERIFIED: internal/app/init.go; VERIFIED: internal/fragments/fragments.go; VERIFIED: internal/releases/releases.go; VERIFIED: internal/render/render.go]

### Pattern 4: Explicit Status and Error Modeling

**What:** Return structured resolution statuses for ordinary outcomes such as `uninitialized`, `legacy_only`, `ambiguous`, `invalid_manifest`, and `resolved`, and reserve returned `error` values for true programmer or IO failures such as unreadable directories, invalid env state that cannot be normalized, or TOML parse failures that the caller cannot classify locally. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/proposals/layout-resolution.md; ASSUMED]

**When to use:** Resolver entry points should classify filesystem evidence into statusful results; higher layers decide whether that status is acceptable for the command being executed. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

**Why it fits the CLI:** The repo already prefers explicit wrapped errors and centralized CLI failure formatting, and Go's `errors.Is` works well when a small number of sentinel errors still matter for transport or boundary failures. [VERIFIED: .planning/codebase/CONVENTIONS.md; VERIFIED: cmd/changes/main.go; CITED: https://pkg.go.dev/errors#Is]

### Anti-Patterns to Avoid

- **Scattered precedence logic:** Do not let `internal/app`, `internal/cli`, or downstream packages read `CHANGES_HOME` or XDG variables directly; Phase 2 needs one shared resolution engine. [VERIFIED: .planning/proposals/layout-resolution.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]
- **String-prefix containment checks:** Do not use `strings.HasPrefix` to prove repo containment; lexical and symlink edge cases make that unsafe. [CITED: https://pkg.go.dev/path/filepath#IsLocal; ASSUMED]
- **Manifest repair on read:** Do not stamp, rewrite, or normalize `layout.toml` during ordinary loads; the approved model allows writes only on init, explicit migration, or explicit repair. [VERIFIED: .planning/proposals/layout-resolution.md]
- **Making `Config.Paths` the new source of truth:** `Config.Paths` is the current compatibility surface, not the future authority for layout selection. [VERIFIED: internal/config/config.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Strict manifest parsing | Custom TOML parser or manual key walking | `toml.DecodeFile` plus `MetaData.Undecoded()` validation. [CITED: https://context7.com/burntsushi/toml/llms.txt] | The repo already uses this pattern in `config.Load`, and it gives schema enforcement without extra parser code. [VERIFIED: internal/config/config.go] |
| Repo-local containment | Prefix checks or raw `..` string rejection | `filepath.Clean`, `filepath.IsLocal`, `filepath.Join`, plus canonicalized existing anchors where equivalence matters. [CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#Join; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED] | This matches Go's documented semantics and prevents obvious repo escapes without inventing a new ruleset. [CITED: https://pkg.go.dev/path/filepath#IsLocal; ASSUMED] |
| Resolution outcome signaling | Error-string matching for normal states | Typed statuses plus wrapped errors and `errors.Is` only for true sentinel cases. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/codebase/CONVENTIONS.md; CITED: https://pkg.go.dev/errors#Is] | The context requires status-rich results, and the repo already treats user-visible errors as wrapped boundary failures. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: internal/cli/app.go] |
| Per-package path logic | New joins in `app`, `fragments`, `releases`, or `render` | One `ResolveAll` engine plus compatibility helpers in `internal/config`. [VERIFIED: internal/fragments/fragments.go; VERIFIED: internal/releases/releases.go; VERIFIED: internal/render/render.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md] | Later `doctor`, ambiguity, and migration work need one evidence model, not four partial implementations. [VERIFIED: .planning/ROADMAP.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md] |

**Key insight:** The hardest Phase 2 problem is not joining directories; it is preserving enough typed evidence and normalization metadata that Phase 3 and Phase 4 can explain, fail, and migrate without re-discovering the filesystem from scratch. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/ROADMAP.md]

## Common Pitfalls

### Pitfall 1: Treating Bootstrap Precedence as an Operational Override

**What goes wrong:** `CHANGES_HOME` silently beats an already-valid XDG candidate, which violates the approved ambiguity and authority model. [VERIFIED: .planning/proposals/layout-resolution.md]

**Why it happens:** It is easy to treat env precedence as if it were the same thing as operational validity, but the proposal separates bootstrap choice from on-disk authority. [VERIFIED: .planning/proposals/layout-resolution.md]

**How to avoid:** Always inspect all supported candidates first, classify them, and only use precedence tables when the scope is uninitialized. [VERIFIED: .planning/proposals/layout-resolution.md; ASSUMED]

**Warning signs:** Resolver wrappers or tests assert env-derived paths without creating or checking competing on-disk candidates. [ASSUMED]

### Pitfall 2: Relying on Lexical Checks Alone for Canonical Equivalence

**What goes wrong:** Two paths that point to the same existing root compare as different, or a symlinked root bypasses naive equivalence logic. [CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED]

**Why it happens:** `filepath.IsLocal` is lexical only and explicitly does not consider symlinks. [CITED: https://pkg.go.dev/path/filepath#IsLocal]

**How to avoid:** Canonicalize only the trusted existing anchors needed for comparison, then derive child paths lexically from those anchors. [CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED]

**Warning signs:** Tests pass for `..` escapes but never cover symlinked repo roots or symlinked `CHANGES_HOME` roots. [ASSUMED]

### Pitfall 3: Rewriting `layout.toml` During Ordinary Loads

**What goes wrong:** Ordinary commands mutate structural metadata, creating churn and violating `MIGR-01`. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md]

**Why it happens:** It is tempting to "repair" or normalize manifests during read time once decode logic exists. [ASSUMED]

**How to avoid:** Keep encode/write functions separate from resolve/load functions and ensure ordinary commands only read manifests. [VERIFIED: .planning/proposals/layout-resolution.md; VERIFIED: internal/config/config.go; ASSUMED]

**Warning signs:** Resolver code opens `layout.toml` with write intent or normalizes symbolic values before returning them. [ASSUMED]

### Pitfall 4: Breaking Current Consumers During the Core Cutover

**What goes wrong:** `init`, fragment creation, releases, or rendering break because helpers disappear before the new resolver is wired beneath them. [VERIFIED: internal/app/init.go; VERIFIED: internal/fragments/fragments.go; VERIFIED: internal/releases/releases.go; VERIFIED: internal/render/render.go]

**Why it happens:** The current codebase has one helper seam, but many downstream callers rely on it. [VERIFIED: internal/config/config.go; VERIFIED: .planning/codebase/STRUCTURE.md]

**How to avoid:** Keep the helper surface stable in Phase 2 and swap its implementation to resolver-backed lookups first. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; ASSUMED]

**Warning signs:** Phase 2 plans include edits across many downstream packages before `internal/config` has stable tests for the new engine. [ASSUMED]

## Code Examples

Verified patterns from official sources and existing repo usage:

### Strict TOML Decode With Unsupported-Key Detection

Source: BurntSushi TOML docs and the repo's existing `config.Load` pattern. [CITED: https://context7.com/burntsushi/toml/llms.txt; VERIFIED: internal/config/config.go]

```go
var manifest LayoutManifest
meta, err := toml.DecodeFile(path, &manifest)
if err != nil {
	return fmt.Errorf("decode layout manifest: %w", err)
}
if undecoded := meta.Undecoded(); len(undecoded) > 0 {
	return fmt.Errorf("decode layout manifest: unsupported keys: %s", joinKeys(undecoded))
}
```

### Repo-Local Containment Guard

Source: `filepath.Clean`, `filepath.IsLocal`, and `filepath.Join` docs. [CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#Join]

```go
func resolveRepoRelative(repoRoot, rel string) (string, error) {
	clean := filepath.Clean(rel)
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("resolver: %q escapes repo root", rel)
	}
	return filepath.Join(repoRoot, clean), nil
}
```

### Status-First Resolver Wrapper

Source: Phase 2 context and existing wrapped-error conventions. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; VERIFIED: .planning/codebase/CONVENTIONS.md; ASSUMED]

```go
func ResolveRepo(ctx ResolveContext) (ScopeResolution, error) {
	all, err := ResolveAll(ctx)
	if err != nil {
		return ScopeResolution{}, fmt.Errorf("resolve repo layout: %w", err)
	}
	return all.Repo, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Repo-local helpers plus `Config.Paths` drive path joins directly in `internal/config`. [VERIFIED: internal/config/config.go] | Resolver-backed typed layout objects should become the authority, with helper functions acting as compatibility adapters only. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; ASSUMED] | Phase 2. [VERIFIED: .planning/ROADMAP.md] | Centralizes precedence, candidate evidence, and future ambiguity handling in one core model. [VERIFIED: .planning/ROADMAP.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md] |
| Global path handling is effectively a single `UserConfigPath(home)` helper today. [VERIFIED: internal/config/config.go] | Phase 2 must resolve full global `config`, `data`, and `state` layouts for both `xdg` and `home`, not isolated strings. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Phase 2. [VERIFIED: .planning/ROADMAP.md] | Unlocks `GLBL-01` and `GLBL-02` without forcing a CLI UX cutover in the same phase. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/ROADMAP.md] |

**Deprecated/outdated:**

- Treating `Config.Paths` as the authority for layout choice is a transitional pattern once the resolver lands. [VERIFIED: internal/config/config.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]
- Ad hoc path joining in consumers is incompatible with the later ambiguity, doctor, and migration phases. [VERIFIED: .planning/ROADMAP.md; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Canonicalize only existing anchor roots with `Abs` plus `EvalSymlinks`, then derive child paths lexically rather than `EvalSymlinks`-ing every descendant path. [ASSUMED] | Architecture Patterns | Candidate equivalence may be under- or over-normalized on symlink-heavy setups. |
| A2 | Keep compatibility wrappers exported in `internal/config` throughout Phase 2 instead of cutting all consumers over to typed resolver results immediately. [ASSUMED] | Architecture Patterns | Phase 2 scope could expand into unnecessary churn if the transition strategy changes. |
| A3 | A small status enum such as `uninitialized`, `resolved`, `legacy_only`, `ambiguous`, and `invalid` is sufficient for Phase 2 and can evolve in Phase 3. [ASSUMED] | Architecture Patterns | Later `doctor` or ambiguity flows may need a richer status matrix than initially planned. |

## Open Questions

1. **Should Phase 2 add a dedicated global-config load API now, or only global path resolution?**
What we know: `GLBL-01` and `GLBL-02` require global path resolution, and the proposal limits bootstrap-affecting global config to `[repo.init]`. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md]
What's unclear: No current command consumes a real global config file yet, so the minimum Phase 2 surface is not forced by current callers. [VERIFIED: internal/cli/app.go; VERIFIED: internal/app/app.go]
Recommendation: Plan for global path resolution and manifest parsing now, but keep global config loading as a narrow helper unless a Phase 2 plan explicitly needs more than `[repo.init]`. [ASSUMED]

2. **Where should deterministic repo-init default selection live?**
What we know: `REPO-03` is a Phase 2 requirement and the proposal defines a precedence table for repo initialization when no layout exists. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md]
What's unclear: The repo currently puts init orchestration in `internal/app/init.go`, while the phase context wants the shared engine to live in `internal/config`. [VERIFIED: internal/app/init.go; VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]
Recommendation: Put the precedence decision function in `internal/config` so it can be reused later by `init`, `init global`, and `doctor` without duplicating policy. [ASSUMED]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Phase 2 implementation and all automated tests | ✓ [VERIFIED: go version] | 1.26.1 installed; module targets 1.26.0. [VERIFIED: go version; VERIFIED: go.mod] | — |
| Git | Existing CLI integration tests that initialize temp repositories | ✓ [VERIFIED: git --version] | 2.53.0. [VERIFIED: git --version] | Package-level resolver tests can still run without Git, but current CLI integration coverage expects it. [VERIFIED: internal/cli/app_integration_test.go; ASSUMED] |

**Missing dependencies with no fallback:**

- None for Phase 2 implementation and test execution on this machine. [VERIFIED: go version; VERIFIED: git --version; VERIFIED: go test ./...]

**Missing dependencies with fallback:**

- None. [VERIFIED: go version; VERIFIED: git --version]

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` package under Go 1.26.x. [VERIFIED: .planning/codebase/TESTING.md; VERIFIED: go version] |
| Config file | None; test execution is command-driven. [VERIFIED: .planning/codebase/TESTING.md] |
| Quick run command | `go test ./internal/config -count=1` [VERIFIED: .planning/codebase/TESTING.md; VERIFIED: AGENTS.md; ASSUMED] |
| Full suite command | `go test ./...` [VERIFIED: .planning/codebase/TESTING.md; VERIFIED: .github/workflows/ci.yml; VERIFIED: go test ./...] |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GLBL-01 | `CHANGES_HOME` beats XDG env vars for global bootstrap preference when no authoritative candidate exists. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Unit | `go test ./internal/config -run TestResolveGlobalPrefersChangesHomeOverXDG -count=1` [ASSUMED] | ❌ Wave 0 |
| GLBL-02 | Global `xdg` and `home` layouts both resolve `config`, `data`, and `state` correctly. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Unit | `go test ./internal/config -run TestResolveGlobalPathsForSupportedStyles -count=1` [ASSUMED] | ❌ Wave 0 |
| REPO-01 | Repo-local `xdg` and `home` layouts both resolve `config`, `data`, and `state` correctly without repo escape. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Unit | `go test ./internal/config -run TestResolveRepoPathsForSupportedStyles -count=1` [ASSUMED] | ❌ Wave 0 |
| REPO-03 | Repo init default selection is deterministic when no repo artifacts exist. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Unit | `go test ./internal/config -run TestSelectRepoInitDefaultDeterministically -count=1` [ASSUMED] | ❌ Wave 0 |
| MIGR-01 | `layout.toml` preserves structural symbolic metadata and ordinary resolution does not rewrite it. [VERIFIED: .planning/REQUIREMENTS.md; VERIFIED: .planning/proposals/layout-resolution.md] | Unit + service regression | `go test ./internal/config -run TestResolveManifestPreservesSymbolicLayoutWithoutRewrite -count=1` [ASSUMED] | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/config -count=1` [VERIFIED: AGENTS.md; ASSUMED]
- **Per wave merge:** `go test ./...` [VERIFIED: .github/workflows/ci.yml; VERIFIED: go test ./...]
- **Phase gate:** Full suite green before `/gsd-verify-work`. [VERIFIED: .planning/config.json]

### Wave 0 Gaps

- [ ] `internal/config/resolution_test.go` — precedence, supported-style path resolution, candidate status, and containment checks for `GLBL-01`, `GLBL-02`, `REPO-01`, and `REPO-03`. [ASSUMED]
- [ ] `internal/config/manifest_test.go` — strict `layout.toml` decoding, symbolic vs resolved path preservation, and no-rewrite behavior for `MIGR-01`. [ASSUMED]
- [ ] Compatibility-wrapper regression coverage in `internal/config/config_test.go` or a new `internal/config/compat_paths_test.go` so existing helpers prove they delegate to the resolver. [ASSUMED]

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no. [VERIFIED: phase scope is local path resolution only in .planning/ROADMAP.md] | Not applicable to this phase. [VERIFIED: .planning/ROADMAP.md] |
| V3 Session Management | no. [VERIFIED: phase scope is local path resolution only in .planning/ROADMAP.md] | Not applicable to this phase. [VERIFIED: .planning/ROADMAP.md] |
| V4 Access Control | no at the account/session level; the relevant boundary is filesystem containment rather than user authorization. [VERIFIED: .planning/ROADMAP.md; ASSUMED] | Enforce repo-root containment and single authoritative target selection inside the resolver. [VERIFIED: .planning/proposals/layout-resolution.md; CITED: https://pkg.go.dev/path/filepath#IsLocal; ASSUMED] |
| V5 Input Validation | yes. [VERIFIED: .planning/proposals/layout-resolution.md] | Strict `layout.toml` decode with unsupported-key rejection, allowed symbolic references only, and path normalization before comparison. [VERIFIED: .planning/proposals/layout-resolution.md; CITED: https://context7.com/burntsushi/toml/llms.txt; CITED: https://pkg.go.dev/path/filepath#Clean] |
| V6 Cryptography | no. [VERIFIED: phase scope and current stack show no crypto requirements in .planning/ROADMAP.md and go.mod] | Not applicable to this phase. [VERIFIED: .planning/ROADMAP.md; VERIFIED: go.mod] |

### Known Threat Patterns for this Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Repo-local path escape via `..` or absolute paths in manifest expansion | Tampering | Reject non-local repo-relative segments with `filepath.IsLocal` before `Join`, and keep allowed symbolic references narrow. [CITED: https://pkg.go.dev/path/filepath#IsLocal; VERIFIED: .planning/proposals/layout-resolution.md] |
| Symlink-induced root confusion during candidate comparison | Tampering | Canonicalize trusted existing anchors for equivalence checks before comparing candidates. [CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; ASSUMED] |
| Malformed or extra manifest keys changing behavior silently | Tampering | Use strict TOML decode with `Undecoded()` rejection and explicit scope/style validation. [CITED: https://context7.com/burntsushi/toml/llms.txt; VERIFIED: .planning/proposals/layout-resolution.md] |
| Wrong write target chosen when multiple candidates exist | Tampering | Preserve all candidate evidence now and fail loudly on ambiguity in Phase 3 instead of silently picking one. [VERIFIED: .planning/proposals/layout-resolution.md; VERIFIED: .planning/ROADMAP.md] |

## Sources

### Primary (HIGH confidence)

- `.planning/phases/02-resolution-core/02-CONTEXT.md` — locked Phase 2 decisions, transition strategy, and evidence/status requirements. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md]
- `.planning/proposals/layout-resolution.md` — authoritative supported layouts, precedence rules, manifest schema, and migration constraints. [VERIFIED: .planning/proposals/layout-resolution.md]
- `.planning/REQUIREMENTS.md` — requirement mapping for `GLBL-01`, `GLBL-02`, `REPO-01`, `REPO-03`, and `MIGR-01`. [VERIFIED: .planning/REQUIREMENTS.md]
- `internal/config/config.go`, `internal/app/init.go`, `internal/app/app.go`, `internal/cli/app.go` — current package seams and path/helper consumers. [VERIFIED: internal/config/config.go; VERIFIED: internal/app/init.go; VERIFIED: internal/app/app.go; VERIFIED: internal/cli/app.go]
- `docs/decisions/ADR-0001-repo-local-xdg-layout.md` — baseline repo-local XDG rationale that Phase 2 must preserve. [VERIFIED: docs/decisions/ADR-0001-repo-local-xdg-layout.md]
- `https://specifications.freedesktop.org/basedir/latest/` — XDG config/data/state definitions and absolute-path requirement. [CITED: https://specifications.freedesktop.org/basedir/latest/]
- `https://pkg.go.dev/os#UserConfigDir` and `https://pkg.go.dev/os#UserHomeDir` — Go stdlib behavior for config and home resolution. [CITED: https://pkg.go.dev/os#UserConfigDir; CITED: https://pkg.go.dev/os#UserHomeDir]
- `https://pkg.go.dev/path/filepath#Clean`, `https://pkg.go.dev/path/filepath#IsLocal`, `https://pkg.go.dev/path/filepath#EvalSymlinks`, `https://pkg.go.dev/path/filepath#Rel`, `https://pkg.go.dev/path/filepath#Join` — Go stdlib path normalization and containment behavior. [CITED: https://pkg.go.dev/path/filepath#Clean; CITED: https://pkg.go.dev/path/filepath#IsLocal; CITED: https://pkg.go.dev/path/filepath#EvalSymlinks; CITED: https://pkg.go.dev/path/filepath#Rel; CITED: https://pkg.go.dev/path/filepath#Join]
- `https://pkg.go.dev/errors#Is` and `https://pkg.go.dev/flag#FlagSet` — Go stdlib error classification and CLI patterns. [CITED: https://pkg.go.dev/errors#Is; CITED: https://pkg.go.dev/flag#FlagSet]
- Context7 `/burntsushi/toml` — strict decode and encode patterns for manifest parsing. [CITED: https://context7.com/burntsushi/toml/llms.txt]
- `go version`, `go list -m -json all`, `git --version`, and `go test ./...` run on 2026-04-06 — verified environment and baseline test health. [VERIFIED: go version; VERIFIED: go list -m -json all; VERIFIED: git --version; VERIFIED: go test ./...]

### Secondary (MEDIUM confidence)

- `.planning/codebase/CONVENTIONS.md`, `.planning/codebase/STRUCTURE.md`, `.planning/codebase/STACK.md`, and `.planning/codebase/TESTING.md` — repo-local architecture and testing norms used to shape recommendations. [VERIFIED: .planning/codebase/CONVENTIONS.md; VERIFIED: .planning/codebase/STRUCTURE.md; VERIFIED: .planning/codebase/STACK.md; VERIFIED: .planning/codebase/TESTING.md]

### Tertiary (LOW confidence)

- None. [VERIFIED: this research used only local docs/code, official specs/docs, Context7, and local command verification]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — existing dependencies and toolchain were verified locally, and the phase does not require a new library decision. [VERIFIED: go.mod; VERIFIED: go version; VERIFIED: go list -m -json all]
- Architecture: MEDIUM — the locked decisions are explicit, but exact resolver type names and file split remain discretionary. [VERIFIED: .planning/phases/02-resolution-core/02-CONTEXT.md; ASSUMED]
- Pitfalls: HIGH — they are driven directly by the approved proposal and Go stdlib path semantics. [VERIFIED: .planning/proposals/layout-resolution.md; CITED: https://pkg.go.dev/path/filepath#IsLocal]

**Research date:** 2026-04-06. [VERIFIED: current date]
**Valid until:** 2026-05-06 for repo-local architecture guidance; re-check external docs sooner only if the Go toolchain or proposal changes. [ASSUMED]
