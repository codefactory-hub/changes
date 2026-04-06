# Phase 1 Requirement and Decision Matrix

## Requirements

| Requirement ID | Artifact File | Exact Section or Command | Status |
| --- | --- | --- | --- |
| CMD-01 | `.planning/proposals/layout-resolution.md` | `changes doctor [--scope global|repo|all] [--explain] [--json]` in `## Approved Command Surface` | Covered |
| CMD-02 | `.planning/phases/01-layout-proposal/01-command-contract.md` | `changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]` in `## Approved Command Lines` | Covered |
| CMD-03 | `.planning/proposals/layout-resolution.md` | `## Bootstrap and Default Precedence`, `## Global Config Shape`, and `## Repo .gitignore Rule` | Covered |
| CMD-04 | `.planning/phases/01-layout-proposal/01-command-contract.md` | `changes init [--layout xdg|home] [--home PATH]` and `changes init global [--layout xdg|home] [--home PATH]` in `## Approved Command Lines` | Covered |
| CMD-05 | `.planning/proposals/layout-resolution.md` | `## Manifest shape` examples and the command blocks in `## Approved Command Surface` | Covered |

## Locked Decisions

| Decision ID | Artifact File | Exact Heading or Rule Text | Status |
| --- | --- | --- | --- |
| D-01 | `.planning/proposals/layout-resolution.md` | ``changes` supports exactly 2 layout styles: `xdg` and `home`` in `## Approved Model` | Locked |
| D-02 | `.planning/proposals/layout-resolution.md` | `Layout resolution happens independently for 2 scopes: global and repo` in `## Approved Model` | Locked |
| D-03 | `.planning/proposals/layout-resolution.md` | `Built-in default locations chooses between supported styles only; it does not define a third style` in `### Notes` | Locked |
| D-04 | `.planning/proposals/layout-resolution.md` | `### Global bootstrap when no global layout exists` | Locked |
| D-05 | `.planning/proposals/layout-resolution.md` | `### Repo initialization when no repo layout exists` | Locked |
| D-06 | `.planning/proposals/layout-resolution.md` | ``CHANGES_HOME` never silently wins over a conflicting valid candidate already present on disk` in `## Ambiguity Rule` | Locked |
| D-07 | `.planning/proposals/layout-resolution.md` | `Layout metadata must be: structural, not historical; symbolic, not fully resolved; low-churn; written only on init, explicit migration, or explicit repair; never rewritten during ordinary command execution` in `## Layout Manifest Schema` | Locked |
| D-08 | `.planning/proposals/layout-resolution.md` | ``[layout]`` examples using `$REPO_ROOT`, `$CHANGES_HOME`, and `$layout.root` in `### Manifest shape` | Locked |
| D-09 | `.planning/proposals/layout-resolution.md` | `A candidate is operationally valid only when ... layout.toml parses successfully ... manifest scope matches ... manifest style matches` in `### Operationally valid candidates` | Locked |
| D-10 | `.planning/proposals/layout-resolution.md` | `Legacy layouts without a manifest are detectable, but they are not operationally valid` in `### Legacy-detected candidates` | Locked |
| D-11 | `.planning/proposals/layout-resolution.md` | `For legacy diagnosis, config.toml is sufficient as an authoritative changes artifact` in `### Legacy-detected candidates` | Locked |
| D-12 | `.planning/proposals/layout-resolution.md` | `Multiple operationally valid candidates for a scope are always an ambiguity error` in `### Authority outcomes` | Locked |
| D-13 | `.planning/proposals/layout-resolution.md` | `changes doctor is the only command that may inspect legacy-detected-only situations` in `### Authority outcomes` | Locked |
| D-14 | `.planning/proposals/layout-resolution.md` | `Manifest stamping or repair is always explicit; ordinary commands must not opportunistically stamp or repair manifests` in `### Authority outcomes` | Locked |
| D-15 | `.planning/phases/01-layout-proposal/01-command-contract.md` | `changes init [--layout xdg|home] [--home PATH]` and `changes init global [--layout xdg|home] [--home PATH]` in `## Approved Command Lines` | Locked |
| D-16 | `.planning/phases/01-layout-proposal/01-command-contract.md` | ``changes doctor` is the approved inspection surface` and `changes doctor --migration-prompt ... generates an LLM-oriented structured brief` in `## Doctor Output Tiers` | Locked |
| D-17 | `.planning/phases/01-layout-proposal/01-command-contract.md` | `default doctor output stays concise`, `--explain is the richer human-oriented tier`, and `--json is structured inspection output` in `## Doctor Output Tiers` | Locked |
| D-18 | `.planning/phases/01-layout-proposal/01-diagnostic-model.md` | `## Default Output` and `## Explain Output` | Locked |
| D-19 | `.planning/proposals/layout-resolution.md` | `The generated migration help is an LLM-oriented structured brief. It must describe the selected source and destination layouts without becoming an executable shell plan` in `## Migration Prompt Requirements` | Locked |
| D-20 | `.planning/phases/01-layout-proposal/01-diagnostic-model.md` | `## Requested Migration`, `## Origin Layout`, `## Destination Layout`, `## Artifact Inventory`, `## Ambiguity and Conflict Notes`, and `## Required Verification` in `## Migration Prompt Sections` | Locked |
| D-21 | `.planning/phases/01-layout-proposal/01-diagnostic-model.md` | `Preserve exactly one authoritative destination. Do not dual-write or keep two live authoritative layouts. Do not convert this brief into destructive automation without explicit operator review.` in `## Safety Rules` | Locked |
| D-22 | `.planning/proposals/layout-resolution.md` | `Global config must not contain values required to discover the global config itself` and `Allowed bootstrap-affecting global config is limited to repo-init defaults` in `## Global Config Shape` | Locked |
| D-23 | `.planning/proposals/layout-resolution.md` | `home is valid only when style = "home"` and `If style = "home" and home is omitted, default to .changes` in `### Validation` | Locked |
| D-24 | `.planning/proposals/layout-resolution.md` | `Repo .gitignore Rule` with `Repo xdg: ignore /.local/state/` and `Repo home: ignore /.changes/state/` | Locked |

## Coverage Notes

Deferred ideas: none.
Open design questions: none.
