# Phase 1: Layout Proposal - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 01-layout-proposal
**Areas discussed:** Candidate validity, Doctor output, Manifest detection and repair, Migration prompt detail

---

## Candidate validity

| Option | Description | Selected |
|--------|-------------|----------|
| Strict manifest-first | Only manifest-backed layouts count as valid candidates | |
| Legacy-aware structural detection | Manifest-backed layouts are valid; legacy layouts can also count if they match a supported shape and contain authoritative `changes` artifacts | ✓ |
| Very loose path existence | Candidate validity is based mostly on directory presence | |

**User's choice:** Legacy-aware structural detection, with `config.toml` alone treated as enough to prove a legacy layout is real enough to diagnose.
**Notes:** Empty directory trees, `.gitkeep`, and `state`-only locations are not enough. This was later tightened so legacy detection is still possible, but ordinary commands must not operate on legacy-only layouts.

---

## Doctor output

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal human output, rich JSON | Keep human output terse and rely on JSON for detail | |
| Rich human output, matching JSON | Always show detailed diagnostics in human output | |
| Tiered output with separate explanation mode | Keep default output concise, use `--explain` for rich diagnostics, and `--json` for structured output | ✓ |

**User's choice:** Tiered output with a quieter default.
**Notes:** Default `doctor` output should stay concise. Rich candidate analysis belongs behind `--explain`. Structured machine-readable analysis belongs behind `--json`.

---

## Manifest detection and repair

| Option | Description | Selected |
|--------|-------------|----------|
| Passive legacy support only | Legacy layouts continue to operate without a manifest | |
| Passive runtime, explicit repair guidance | Legacy layouts operate normally but doctor recommends repair | |
| Manifest-backed-only normal operation | Only manifest-backed layouts are operationally valid; `doctor` remains available for repair and inspection | ✓ |

**User's choice:** Manifest-backed-only normal operation, with `doctor` as the inspection/repair path.
**Notes:** Ordinary commands should fail when they only find legacy support. `doctor` is the one exception that may inspect those situations and help the user choose or repair.

---

## Migration prompt detail

| Option | Description | Selected |
|--------|-------------|----------|
| High-level advisory only | Short prompt with broad migration guidance | |
| Structured migration brief | Structured prompt with source/destination facts, constraints, and verification expectations | ✓ |
| Near-procedural playbook | Highly prescriptive migration instructions | |

**User's choice:** Structured migration brief.
**Notes:** The prompt should require a structured answer shape with analysis, plan, and verification. It should not ask for concrete shell commands or file operations.

---

## the agent's Discretion

- Exact naming of internal resolver types and helper functions
- Exact formatting of `doctor --json` output as long as it preserves the approved data

## Deferred Ideas

None
