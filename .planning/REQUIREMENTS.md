# Requirements: changes

**Defined:** 2026-04-07
**Milestone:** `0.1.0-rc.2`
**Core Value:** `changes` must make release metadata location predictable, inspectable, and safe even when multiple supported storage layouts are possible.

## rc.2 Requirements

### Legacy Layout Repair

- [ ] **REPAIR-01**: `changes doctor` can repair a legacy repo-local layout by writing the authoritative layout manifest for the preferred supported candidate
- [ ] **REPAIR-02**: `changes doctor --scope repo --repair` refuses to proceed when repo authority is ambiguous instead of guessing
- [ ] **REPAIR-03**: successful repair leaves the repo operational for ordinary commands without requiring manual manifest creation
- [ ] **REPAIR-04**: repair output explains what was repaired and which authoritative layout is now active

### Safety and Scope

- [ ] **SAFE-01**: repair only stamps or repairs one authoritative layout; it does not migrate data or dual-write
- [ ] **SAFE-02**: repair preserves the existing repo-local state ignore rule for the authoritative layout
- [ ] **SAFE-03**: documentation explains when to use repair versus migration prompts

## Out of Scope

| Feature | Reason |
|---------|--------|
| Automatic data migration between `xdg` and `home` layouts | This milestone is about stamping or repairing authoritative manifests, not moving operator data |
| Global-layout repair automation | Repo-local repair is the immediate operator pain point and the fastest narrow fix |
| Directory schema version 2 design | Still deferred behind the immediate legacy-repair gap |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| REPAIR-01 | Phase 6 | Planned |
| REPAIR-02 | Phase 6 | Planned |
| REPAIR-03 | Phase 6 | Planned |
| REPAIR-04 | Phase 6 | Planned |
| SAFE-01 | Phase 6 | Planned |
| SAFE-02 | Phase 6 | Planned |
| SAFE-03 | Phase 6 | Planned |

---
*Requirements defined: 2026-04-07 for `0.1.0-rc.2`*
