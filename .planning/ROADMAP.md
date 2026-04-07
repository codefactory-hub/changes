# Roadmap: changes

## Current Milestone: v0.1.0-rc.2 Legacy Layout Repair

## Overview

This milestone closes the most immediate operator gap left by `0.1.0-rc.1`: existing legacy repos can be diagnosed, but they still require manual manifest creation. The goal is to add an explicit repo-local repair path that stamps the authoritative manifest safely and leaves the repo operational without reopening broader migration semantics.

## Phases

- [ ] **Phase 6: Legacy Layout Repair** - Add a narrow repair command flow for repo-local legacy layouts and verify the repaired repo becomes operational without manual manifest edits

## Phase Details

### Phase 6: Legacy Layout Repair
**Goal**: Add an operator-friendly repo-local repair flow for legacy layouts while preserving authority and single-target safety guarantees
**Depends on**: Shipped milestone `0.1.0-rc.1`
**Requirements**: [REPAIR-01, REPAIR-02, REPAIR-03, REPAIR-04, SAFE-01, SAFE-02, SAFE-03]
**Success Criteria** (what must be TRUE):
  1. A legacy repo-local layout can be repaired without manual manifest authoring
  2. Repair fails loudly on ambiguity and does not migrate or dual-write data
  3. After repair, ordinary commands operate against the authoritative repo-local layout
**Plans**: 0 plans

## Shipped Milestones

- [x] [v0.1.0-rc.1: Flexible Layout Resolution](milestones/v0.1.0-rc.1-ROADMAP.md) — 5 phases, 10 plans, shipped 2026-04-07

## Progress

**Execution Order:**
Phase 6 executes after the shipped `0.1.0-rc.1` milestone.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 6. Legacy Layout Repair | 0/0 | Not started | — |
