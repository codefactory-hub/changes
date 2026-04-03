# PLN-0003 Consistency Fixups For Release Record Model

## Scope

Tighten the repository's live contract so active plans, ADRs, README text, CLI help, tests, and user-facing errors all describe the current `ReleaseRecord` / `ReleaseBundle` model consistently.

This cleanup also broadens the ignored transient-state boundary from `/.local/state/changes/` to all of `/.local/state/` and relocates the repo-local collection catalog into ignored local state.

## Planned changes

- update live docs to describe release records, release bundles, and editorial packages using the accepted current vocabulary
- remove stale references to release manifests, `target_version`, stored `channel`, and older slice-oriented terms from active governed docs
- keep collection snapshot `manifest.json` terminology where it is accurate for devtools collection snapshots
- rename user-facing devtools report fields and messages from manifest wording to release-record wording
- broaden `.gitignore` and initialization behavior to ignore all of `/.local/state/`
- update README examples to point local collection inputs at `.local/state/`
- move the repo-local `catalog.toml` into ignored local state

## Constraints

- do not rename or move governed historical docs unless separately approved
- do not rewrite untracked research drafts as part of this pass
- keep behavior changes minimal; this is primarily a consistency cleanup over the already-implemented release-record model

## Execution notes

- patch active docs first so the repository's written contract matches the implementation
- then patch user-facing strings, report fields, and tests to match the updated contract
- use `git-workflow` for cohesive commits when the cleanup is ready to checkpoint
