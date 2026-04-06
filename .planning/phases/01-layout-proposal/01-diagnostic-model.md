# Phase 1 Diagnostic Model

## Candidate States

The approved per-scope candidate states are:

- `authoritative`: exactly one operationally valid candidate exists for the scope
- `ambiguous`: more than one operationally valid candidate exists for the scope
- `legacy-detected`: no operationally valid candidate exists, but one or more supported-shape legacy candidates with authoritative `changes` artifacts were found
- `uninitialized`: no operationally valid or legacy-detected candidate exists for the scope

Operationally valid means `layout.toml` parses and the manifest `scope` and `style` match the candidate being evaluated. Legacy-detected candidates remain diagnosable, but only `changes doctor` may inspect them for ordinary operator workflows.

- Scopes resolved independently: global and repo.
- Operational validity requires parseable layout.toml with matching scope and style.
- Legacy-only detection is doctor-visible but invalid for ordinary commands.
- Multiple supported candidates = ambiguity error.

## Default Output

Default `changes doctor` output stays concise. It reports only the minimum operator-facing facts needed to understand the current state:

- requested scope
- final status for each inspected scope
- selected style and selected root when the status is `authoritative`
- whether the operator must resolve ambiguity or initialize a layout

Default output does not dump full candidate inventories or precedence traces unless the operator asks for a richer tier.

Doctor tiers: default concise, --explain rich, --json structured.

## Explain Output

`changes doctor --explain` is the richer human-oriented tier. It may include:

- the precedence inputs that influenced bootstrap or preference
- every discovered candidate and why it was or was not operationally valid
- ambiguity reasoning and candidate comparison notes
- repair hints that preserve one authoritative destination

`--explain` must stay descriptive and non-destructive. It does not stamp manifests, repair layouts, or move files.

## JSON Shape

`changes doctor --json` returns structured inspection output with these top-level keys:

- `requested_scope`
- `generated_at`
- `global`
- `repo`
- `summary`

Each per-scope object uses these keys:

- `status`
- `selected_style`
- `selected_root`
- `precedence_inputs`
- `candidates`
- `repair_hint`

Illustrative contract:

```json
{
  "requested_scope": "all",
  "generated_at": "2026-04-06T00:00:00Z",
  "global": {
    "status": "authoritative",
    "selected_style": "home",
    "selected_root": "/home/operator/.changes-home",
    "precedence_inputs": [
      "CHANGES_HOME",
      "built-in default locations"
    ],
    "candidates": [
      {
        "scope": "global",
        "style": "home",
        "root": "/home/operator/.changes-home",
        "state": "authoritative"
      }
    ],
    "repair_hint": ""
  },
  "repo": {
    "status": "uninitialized",
    "selected_style": "",
    "selected_root": "",
    "precedence_inputs": [
      "built-in default locations"
    ],
    "candidates": [],
    "repair_hint": "Run changes init [--layout xdg|home] [--home PATH]."
  },
  "summary": {
    "status_counts": {
      "authoritative": 1,
      "ambiguous": 0,
      "legacy-detected": 0,
      "uninitialized": 1
    }
  }
}
```

The exact nested candidate fields may evolve in implementation, but these top-level and per-scope keys are locked by Phase 1.

## Migration Prompt Sections

`changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]` emits a structured Markdown brief with these exact headings:

## Requested Migration

State the requested scope, requested destination style, optional destination home path, and the command inputs that produced the brief.

## Origin Layout

Describe the currently diagnosed origin layout, including status, style, selected root, manifest presence, and why it is the origin candidate.

## Destination Layout

Describe the requested destination layout, including the exact destination root, config path, data path, and state path that the external tool must preserve.

## Artifact Inventory

Summarize the authoritative artifacts found in config, data, and state, including counts or file inventories gathered by `doctor`.

## Ambiguity and Conflict Notes

List competing supported candidates, legacy-detected candidates, manifest mismatches, or other facts that affect safe migration planning.

## Required Verification

List the checks the external tool must ask the operator to confirm after migration, including manifest validity, authoritative root selection, and expected repo ignore rules.

## Safety Rules

Preserve exactly one authoritative destination.

Do not dual-write or keep two live authoritative layouts.

Do not convert this brief into destructive automation without explicit operator review.

Migration prompt is an advisory Markdown brief with required verification and explicit no-dual-write instructions.
