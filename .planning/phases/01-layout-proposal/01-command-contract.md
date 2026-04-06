# Phase 1 Command Contract

## Approved Command Lines

```text
changes init [--layout xdg|home] [--home PATH]
changes init global [--layout xdg|home] [--home PATH]
changes doctor [--scope global|repo|all] [--explain] [--json]
changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]
```

These command lines are the only approved Phase 1 contract for layout initialization, inspection, and migration guidance. Future implementation work must use these exact command names and flag shapes.

## Locked Rule Summary

- Supported styles: xdg and home only.
- Scopes resolved independently: global and repo.
- Global bootstrap precedence: flags > CHANGES_HOME > XDG env vars > built-in default locations.
- Repo init precedence: flags > [repo.init] defaults > CHANGES_HOME signal > XDG env signal > built-in default locations.
- Doctor tiers: default concise, --explain rich, --json structured.
- Migration prompt is an advisory Markdown brief with required verification and explicit no-dual-write instructions.
- Global config bootstrap keys are limited to [repo.init].
- Repo state ignore rules are /.local/state/ and /.changes/state/.

## Initialization Rules

- `changes init is repo-local only`
- `changes init global is global only`
- `--home is valid only with --layout home`
- `changes init` and `changes init global` support only `xdg` and `home`
- `changes init` may use repo init defaults only from `[repo.init]`
- `changes init` must not introduce a separate `layout` command family or alias

## Repo Initialization Default Sources

When no repo layout exists yet, repo initialization may draw defaults only from the approved precedence model:

1. Explicit command flags
2. Global config repo-init defaults
3. `CHANGES_HOME` as a style preference signal
4. XDG env vars as a style preference signal
5. Built-in default locations

The only allowed global bootstrap-affecting config for repo init defaults is:

```toml
[repo.init]
style = "xdg"
```

or

```toml
[repo.init]
style = "home"
home = ".changes"
```

Validation rules:

- `style` must be `xdg` or `home`
- `home` is valid only when `style = "home"`
- if `style = "home"` and `home` is omitted, `.changes` is the default
- if `style = "xdg"` and `home` is present, initialization must fail

## Doctor Output Tiers

`changes doctor` is the approved inspection surface.

- default `doctor` output stays concise
- `--explain` is the richer human-oriented tier
- `--json` is structured inspection output
- `changes doctor --migration-prompt ...` generates an LLM-oriented structured brief
- `doctor` owns ambiguity inspection, legacy-only inspection, and migration guidance
- Legacy-only detection is doctor-visible but invalid for ordinary commands.

Ordinary commands must not inspect or normalize legacy-only states on their own. When only legacy-detected layouts exist, operators are directed to `changes doctor`.

## Scope and Validation Rules

- `changes doctor` may inspect `global`, `repo`, or `all`
- `changes doctor --migration-prompt` requires `--scope global|repo`
- `changes doctor --migration-prompt` requires `--to xdg|home`
- `--home` on migration prompting is valid only when the destination layout is `home`
- command validation must fail loudly on unsupported styles, unsupported scopes, and invalid flag combinations

## Repo Hygiene Rules

Repo-local initialization and migration must keep the active repo-local state directory ignored:

- `xdg` repositories use `/.local/state/`
- `home` repositories use `/.changes/state/`
- Repo state ignore rules are `/.local/state/` and `/.changes/state/`.

These ignore rules apply to the authoritative repo-local layout only.

## Non-Approved Variants

The following are explicitly out of contract:

- any `changes layout ...` namespace
- any third layout style beyond `xdg` and `home`
- any dual-write or multi-target initialization behavior
- any implicit `--home` behavior when `--layout` is not `home`
