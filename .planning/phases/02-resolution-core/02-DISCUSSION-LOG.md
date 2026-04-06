# Phase 2: Resolution Core - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-06
**Phase:** 02-resolution-core
**Areas discussed:** Resolver API shape, Manifest handling in the core, Legacy detection boundary, Consumer migration strategy, Core result shape, Error and status boundary, Path normalization strictness

---

## Resolver API shape

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, explicit layout objects should be primary | Make explicit resolved layout objects the primary API and architectural model | ✓ |
| No, keep helper-style APIs primary | Preserve helper-style path APIs as the main public surface | |
| Mixed: explicit core, compatibility wrappers during transition | Keep an explicit internal core but treat wrappers as co-primary during migration | |

**User's choice:** `Yes, explicit layout objects should be primary`
**Notes:** The user wanted the Phase 2 core to establish the explicit layout-object model directly instead of hiding the new authority model behind helper-style APIs.

---

## Manifest handling in the core

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, full manifest semantics belong in the core | Parse manifests, validate scope and style, and expand symbolic paths in Phase 2 | ✓ |
| No, parse-only for now | Read raw manifest data but defer most interpretation to later phases | |
| Partial semantics only | Parse and validate some fields now, but defer full path expansion | |

**User's choice:** `Yes, full manifest semantics belong in the core`
**Notes:** The user wanted Phase 2 to own manifest parsing, validation, and symbolic-to-resolved path expansion rather than calling it a resolution core while deferring real resolution semantics.

---

## Legacy detection boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence-rich candidate records | Return all detected candidates with supporting evidence from the Phase 2 core | ✓ |
| Minimal valid or legacy results only | Return only coarse valid or legacy results and defer richer evidence | |

**User's choice:** `Evidence-rich candidate records`
**Notes:** The user wanted Phase 2 to return rich candidate evidence so later ambiguity and `doctor` work would not need to reconstruct filesystem facts.

---

## Consumer migration strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Hard cutover now | Move current consumers directly to the new resolver objects in Phase 2 | |
| Use compatibility wrappers during transition | Keep wrappers that delegate to the new resolver core while consumers migrate intentionally | ✓ |
| Build the core first and defer most rewiring | Implement the core in isolation and delay most consumer changes | |

**User's choice:** `Use compatibility wrappers during transition`
**Notes:** The user preferred to keep Phase 2 focused on core correctness while avoiding unnecessary churn in unrelated packages.

---

## Core result shape

| Option | Description | Selected |
|--------|-------------|----------|
| ResolveAll only | Expose only one top-level all-scopes resolution result | |
| Separate scope APIs only | Expose only `ResolveGlobal` and `ResolveRepo` style entry points | |
| Both | Expose `ResolveAll` plus `ResolveGlobal` and `ResolveRepo` | ✓ |

**User's choice:** `Both`
**Notes:** The user accepted a combined all-scopes result alongside per-scope convenience entry points.

---

## Error and status boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Structured statuses only | The core returns statuses and leaves fatal-versus-acceptable handling to callers | ✓ |
| Hard errors in ordinary resolver APIs | The core immediately errors on ambiguity, legacy-only, or uninitialized states | |
| Both: status core plus operational wrappers | Provide both structured status output and hard-error wrappers | |

**User's choice:** `Structured statuses only`
**Notes:** The user preferred the core to return structured statuses and leave policy decisions to higher-level callers.

---

## Path normalization strictness

| Option | Description | Selected |
|--------|-------------|----------|
| Strict normalization | Normalize aggressively, enforce containment, and canonicalize equivalent paths before comparison | ✓ |
| Minimal normalization | Only do basic cleanup and defer more interpretation | |
| Strict for repo, lighter for global | Apply stronger normalization only to repo-local paths | |

**User's choice:** `Strict normalization`
**Notes:** The user wanted path normalization to be treated as a safety property, including repo containment and equivalent-candidate normalization.

---

## the agent's Discretion

- Exact exported type names for the resolver objects, scope results, and candidate evidence records
- Exact helper decomposition inside `internal/config/`

## Deferred Ideas

None.
