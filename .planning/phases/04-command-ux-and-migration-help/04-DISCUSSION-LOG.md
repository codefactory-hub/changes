# Phase 4: Command UX and Migration Help - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 04-command-ux-and-migration-help
**Areas discussed:** doctor command shape, migration prompt output behavior, init/init global layout flags, documentation shape and examples

---

## Doctor command shape

| Option | Description | Selected |
|--------|-------------|----------|
| Flags only, default scope repo when in a repo | Keep `doctor` as one flag-driven command family; use repo as the default scope when a repo root is available | ✓ |
| Flags only, default scope all | Keep one flag-driven command but always inspect all scopes by default | |
| Different shape | Change the command grammar away from the approved flag-only design | |

**User's choice:** Flags only, default scope repo when in a repo  
**Notes:** The approved Phase 1 command family remains intact. Explicit `--scope global|repo|all` stays available, and migration help remains `doctor --migration-prompt`.

---

## Migration prompt output behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Stdout by default, optional `--output PATH` | Print the generated Markdown brief to stdout by default and optionally write it to a file | ✓ |
| Always write a file | Require file creation for every migration prompt invocation | |
| Different shape | Use a different delivery model for the generated prompt | |

**User's choice:** Stdout by default, optional `--output PATH`  
**Notes:** When `--output PATH` is used, the prompt body should go to the file and stdout may carry only a concise success line.

---

## Init and init global layout flags

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal flags, explicit success output, mention `.gitignore` only when changed | Keep `--layout` and `--home`, show selected layout plus resolved paths, and mention ignore updates only when they were written | ✓ |
| Minimal flags, terse success output | Keep the same flags but keep init output short and omit most path detail | |
| Different shape | Introduce a different user-facing init surface | |

**User's choice:** Minimal flags, explicit success output, mention `.gitignore` only when changed  
**Notes:** The minimal flag surface remains preferred. The user wants operator-visible confirmation of selected config/data/state paths without constant `.gitignore` noise.

---

## Documentation shape and examples

| Option | Description | Selected |
|--------|-------------|----------|
| Lead with defaults, then alternatives and examples | Start docs with the default path and then explain overrides, alternatives, and examples | ✓ |
| Lead with a comparison table | Start docs with a side-by-side matrix of layout styles and scopes | |
| Different shape | Use another doc organization model | |

**User's choice:** Lead with defaults, then alternatives and examples  
**Notes:** This is the preferred starting point for now and may be revisited later if experience shows a better documentation shape is needed.

---

## the agent's Discretion

- Exact wording of help text and success lines
- Exact placement of examples between README and command help, so long as the default-first doc shape is preserved

## Deferred Ideas

None — discussion stayed within phase scope
