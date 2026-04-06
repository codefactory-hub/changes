# Phase 3: Authority and Safety - Discussion Log

**Date:** 2026-04-06
**Phase:** 3
**Status:** Complete

## Areas Discussed

### 1. Ambiguity diagnostics

**Question:** How much detail should ordinary ambiguity errors include?

**Options presented:**
- Terse only
- Moderately explanatory
- Rich inline diagnostics

**User selection:** Terse only

**Captured decision:** Ordinary ambiguity errors stay terse and defer detailed inspection to `changes doctor`.

### 2. Authority error model

**Question:** How should Phase 3 treat authority/safety failures at the user-facing layer?

**Options presented:**
- One generic authority error
- Distinct user-facing error classes
- Partial merge of some classes

**User selection:** Distinct user-facing error classes

**Captured decision:** `ambiguous`, `legacy_only`, `invalid_manifest`, and `uninitialized` remain distinct user-facing failure classes.

### 3. Schema/version compatibility

**Question:** How strict should `schema_version` compatibility be?

**Options presented:**
- Strict exact-version support
- Forward-compatible when possible
- Mixed policy

**User selection:** Strict exact-version support

**Captured decision:** Only exact supported schema versions are operationally valid in Phase 3.

### 4. Single-target write gating

**Question:** How should the no-dual-write rule be enforced?

**Options presented:**
- Per write entry point
- One centralized write API only
- Shared authority-check helper called by each write entry point

**User selection:** Per write entry point

**Captured decision:** Each write-capable operation must enforce authority before writing.

### 5. Mixed-candidate tolerance

**Question:** If one candidate is operationally valid while sibling candidates are `invalid_manifest` or legacy-only, should ordinary commands proceed or fail?

**Options discussed:**
- Proceed if exactly one candidate is operationally valid
- Fail if any other supported-shape candidate exists
- Mixed behavior by sibling status

**User response:** Proceed when there is one valid candidate, but warn so `doctor` can clean it up; no suppression config in Phase 3.

**Captured decision:** One valid candidate may operate, non-operational siblings warn and point to `doctor`, and warning suppression is explicitly out of scope for Phase 3.

### 6. Explicit-command exceptions

**Question:** Should `init` get special authority exceptions in messy states?

**Options presented:**
- No exceptions
- Limited `init` exception
- Broad setup exception

**User selection:** No exceptions

**Captured decision:** `init` obeys the same authority rules as other commands in Phase 3.

### 7. Equivalent-candidate collapse

**Question:** If two candidates canonicalize to the same physical location, should they count as one authority target or ambiguity?

**Options presented:**
- Collapse to one authority target
- Keep as ambiguity anyway
- Collapse only in narrow cases

**User selection:** Collapse to one authority target

**Captured decision:** Canonically equivalent candidates do not create false ambiguity.

### 8. Warning surface

**Question:** Where should non-blocking sibling-candidate warnings appear?

**Options presented:**
- CLI warning only, on stderr
- Inline service-layer text
- Silent in normal commands, visible only in `doctor`

**User selection:** CLI warning only, on stderr

**Captured decision:** Lower layers return structured warning information and the CLI prints warnings on stderr.

### 9. Terse ambiguity contents

**Question:** What is the minimum ambiguity error content?

**Options presented:**
- Just “ambiguous layout”
- Scope + short failure + next command
- Scope + short failure + candidate count + next command

**User selection:** Scope + short failure + next command

**Captured decision:** Terse ambiguity errors must identify the scope and tell the user what `changes doctor` command to run next.

### 10. Warning scope across ordinary commands

**Question:** Should the warning appear for every ordinary command that resolves the affected scope, or only for some?

**Options presented:**
- Every ordinary command that resolves the affected scope
- Only write-affecting commands
- Only some human-facing commands

**User selection:** Every ordinary command that resolves the affected scope

**Captured decision:** Warning behavior should be consistent across ordinary commands, not selectively hidden.

## Deferred Ideas

- Per-path warning suppression config for invalid or legacy sibling candidates
- Rich inline diagnostics in ordinary command errors
- Phase 4 `doctor` explanation and migration UX details

---
*Discussion captured: 2026-04-06*
