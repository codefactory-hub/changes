# Layout Resolution Proposal

**Status:** Approved for implementation planning on 2026-04-06  
**Scope:** Global and repo-local storage layout resolution for `changes`  
**Implementation rule:** Any implementation work for this project must conform to this proposal unless this document is amended first.

## Purpose

Define the authoritative design for flexible storage layout resolution in `changes` before implementation begins. This proposal locks the supported layout styles, precedence rules, config shape, manifest schema, command surface, and migration assistance model.

## Approved Model

- `changes` supports exactly 2 layout styles: `xdg` and `home`
- Layout resolution happens independently for 2 scopes: `global` and `repo`
- Each scope resolves to exactly one authoritative layout for ordinary operation
- Reads and writes must always go through the resolved authoritative layout object
- Writes target exactly one authoritative layout; no dual-write behavior is allowed
- `CHANGES_HOME` can influence bootstrap and default selection, but it never silently beats a conflicting valid on-disk candidate

## Locked Phase 2 Rules

- Supported styles: xdg and home only.
- Scopes resolved independently: global and repo.
- Global bootstrap precedence: flags > CHANGES_HOME > XDG env vars > built-in default locations.
- Repo init precedence: flags > [repo.init] defaults > CHANGES_HOME signal > XDG env signal > built-in default locations.
- Operational validity requires parseable layout.toml with matching scope and style.
- Legacy-only detection is doctor-visible but invalid for ordinary commands.
- Multiple supported candidates = ambiguity error.
- Repair or manifest stamping is explicit only.
- Doctor tiers: default concise, --explain rich, --json structured.
- Migration prompt is an advisory Markdown brief with required verification and explicit no-dual-write instructions.
- Global config bootstrap keys are limited to [repo.init].
- Repo state ignore rules are /.local/state/ and /.changes/state/.

## Supported Layouts

### Global `xdg`

- Config directory is derived from XDG config semantics
- Data directory is derived from XDG data semantics
- State directory is derived from XDG state semantics

### Global `home`

- Root directory comes from `CHANGES_HOME`
- Config lives under the home layout root
- Data lives under the home layout root
- State lives under the home layout root

### Repo `xdg`

- Config: `.config/changes`
- Data: `.local/share/changes`
- State: `.local/state/changes`

### Repo `home`

- Root: `.changes`
- Config: `.changes/config`
- Data: `.changes/data`
- State: `.changes/state`

## Runtime Resolution Rules

For each scope (`global`, `repo`):

1. Inspect all supported candidates for that scope
2. If exactly one valid candidate exists, use it
3. If multiple valid candidates exist, fail with an ambiguity error
4. If no valid candidate exists, treat the scope as uninitialized and continue according to command semantics

The ambiguity rule applies even if one candidate is derived from `CHANGES_HOME` and another is discoverable through XDG paths. Environment signals influence preference and bootstrapping, but they do not silently override a conflicting real on-disk situation.

## Candidate Validity and Authority

### Operationally valid candidates

A candidate is operationally valid only when all of the following are true:

1. The candidate matches one of the supported layout shapes for its scope
2. `layout.toml` exists in the candidate config directory
3. `layout.toml` parses successfully
4. The manifest `scope` matches the candidate scope
5. The manifest `style` matches the candidate style

Ordinary commands may operate only on operationally valid candidates.

### Legacy-detected candidates

Legacy layouts without a manifest are detectable, but they are not operationally valid.

A legacy candidate is diagnosable only when it:

1. Matches a supported layout shape for the requested scope and style
2. Contains an authoritative `changes` artifact inside that shape

For legacy diagnosis, `config.toml` is sufficient as an authoritative `changes` artifact.

### Authority outcomes

- Exactly one operationally valid candidate for a scope is `authoritative`
- Multiple operationally valid candidates for a scope are always an ambiguity error
- A scope with only legacy-detected candidates is not operationally valid for ordinary commands
- `changes doctor` is the only command that may inspect legacy-detected-only situations
- Manifest stamping or repair is always explicit; ordinary commands must not opportunistically stamp or repair manifests

## Bootstrap and Default Precedence

### Global bootstrap when no global layout exists

1. Explicit command flags
2. `CHANGES_HOME`
3. XDG env vars
4. Built-in default locations

### Repo initialization when no repo layout exists

1. Explicit command flags
2. Global config repo-init defaults
3. `CHANGES_HOME` as a style preference signal
4. XDG env vars as a style preference signal
5. Built-in default locations

### Notes

- “Built-in default locations” chooses between supported styles only; it does not define a third style
- Default repo behavior remains `xdg` unless a stronger approved input selects `home`

## Ambiguity Rule

If multiple supported layouts exist for the same scope:

- `changes` must fail
- The error must identify each candidate and why it was considered valid
- The error must explain that the user must pick one authoritative location
- The error may suggest which candidate appears strongest based on diagnostic heuristics
- The error may suggest generating a migration prompt to merge into one authoritative destination
- `CHANGES_HOME` never silently wins over a conflicting valid candidate already present on disk

## Recommended Candidate Heuristics

These heuristics are diagnostic only and never drive automatic selection:

1. Prefer a candidate with a valid layout manifest over one without
2. Prefer a candidate with a newer supported `schema_version`
3. Prefer a candidate with a more complete `changes` directory set and more actual artifacts
4. Prefer a candidate with clearer structural integrity when the above are tied

## Layout Manifest Schema

Layout metadata must be:

- structural, not historical
- symbolic, not fully resolved
- low-churn
- written only on `init`, explicit migration, or explicit repair
- never rewritten during ordinary command execution

### Manifest placement

- Stored in the config directory of the authoritative layout
- Repo `xdg`: `.config/changes/layout.toml`
- Repo `home`: `.changes/config/layout.toml`
- Global `xdg`: global config directory `layout.toml`
- Global `home`: `$CHANGES_HOME/config/layout.toml`

### Manifest shape

Global `home` example:

```toml
schema_version = 1
scope = "global"
style = "home"

[layout]
root = "$CHANGES_HOME"
config = "$layout.root/config"
data = "$layout.root/data"
state = "$layout.root/state"
```

Repo `home` example:

```toml
schema_version = 1
scope = "repo"
style = "home"

[layout]
root = "$REPO_ROOT/.changes"
config = "$layout.root/config"
data = "$layout.root/data"
state = "$layout.root/state"
```

Repo `xdg` example:

```toml
schema_version = 1
scope = "repo"
style = "xdg"

[layout]
root = "$REPO_ROOT"
config = "$REPO_ROOT/.config/changes"
data = "$REPO_ROOT/.local/share/changes"
state = "$REPO_ROOT/.local/state/changes"
```

### Allowed symbolic references

- `$REPO_ROOT`
- `$CHANGES_HOME`
- `$XDG_CONFIG_HOME`
- `$XDG_DATA_HOME`
- `$XDG_STATE_HOME`
- `$HOME`
- `$layout.root`

No version stamps, migration timestamps, or last-written markers are part of the initial schema.

Operational validity requires that `layout.toml` parse and that its `scope` and `style` match the candidate being evaluated. Legacy-detected layouts without a manifest remain diagnosable, but they are not valid for ordinary operation.

## Repo `.gitignore` Rule

Repo-local initialization and migration must ensure the authoritative repo-local `state` directory is ignored consistently.

- Repo `xdg`: ignore `/.local/state/`
- Repo `home`: ignore `/.changes/state/`

This applies only to the active authoritative repo layout.

## Global Config Shape

Global config must not contain values required to discover the global config itself.

Allowed bootstrap-affecting global config is limited to repo-init defaults:

```toml
[repo.init]
style = "home"
home = ".changes"
```

or

```toml
[repo.init]
style = "xdg"
```

### Validation

- `style` must be `xdg` or `home`
- `home` is valid only when `style = "home"`
- If `style = "home"` and `home` is omitted, default to `.changes`
- If `style = "xdg"` and `home` is present, that is a configuration error
- No bootstrap-affecting keys outside `[repo.init]` are allowed

## Approved Command Surface

### Initialization

```text
changes init [--layout xdg|home] [--home PATH]
changes init global [--layout xdg|home] [--home PATH]
```

Rules:

- `--home` is only valid with `--layout home`
- `changes init` is repo-local only
- `changes init global` is global only

### Diagnosis and migration assistance

```text
changes doctor [--scope global|repo|all] [--explain] [--json]
changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]
```

Rules:

- `doctor` owns inspection, ambiguity diagnostics, and migration guidance
- Default `doctor` output stays concise
- Richer candidate analysis belongs behind `--explain`
- `--json` is structured inspection output
- Migration prompt generation belongs to `doctor`, not a separate `layout` namespace
- `doctor` must be able to explain why a scope resolved, failed, or is ambiguous
- Ordinary commands fail when they find only `legacy-detected` layouts; only `doctor` may inspect those states

## Migration Prompt Requirements

`changes doctor --migration-prompt ...` must generate a Markdown prompt suitable for an external LLM tool. The prompt must include:

- exact origin and destination paths
- origin and destination layout style
- detected manifest presence and `schema_version`
- file inventories or artifact counts for config, data, and state
- ambiguity/conflict notes if multiple candidates exist
- instructions to preserve one authoritative destination only
- an explicit prohibition on dual-write outcomes
- a required verification section

The generated migration help is an LLM-oriented structured brief. It must describe the selected source and destination layouts without becoming an executable shell plan.

The migration prompt is advisory. It does not perform migration by itself.

## Non-Goals

- No automatic dual-write synchronization
- No silent conflict resolution between competing layouts
- No automatic destructive migration
- No third layout style beyond `xdg` and `home`

## Implementation Gate

Before implementation begins:

1. This proposal remains the governing design artifact
2. Phase 2 work must reference this document
3. Any design changes must update this document first
