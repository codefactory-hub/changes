# Phase 5 Discussion Log

## Summary

Phase 5 was framed as a rollout-verification phase rather than a new feature phase. The discussion focused on what must be proven about default behavior, compatibility, precedence, and legacy-repo handling before closing the milestone.

## Decisions

1. Existing-repo compatibility uses a mixed interpretation.
   - Manifest-backed repos must continue to operate normally.
   - Legacy repos without manifests do not need to operate normally.
   - The phase must verify that legacy repos fail cleanly and are diagnosable through `doctor`.

2. Default-selection verification uses a focused precedence set.
   - Plain repo init defaults to repo `xdg`.
   - Plain global init defaults to global `xdg`.
   - Repo init respects `[repo.init]` defaults.
   - `CHANGES_HOME` is a repo-style preference signal.
   - `CHANGES_HOME` beats XDG env for global init preference.
   - Explicit flags beat other sources.
   - Invalid `style` / `home` combinations remain rejected.

3. Verification depth uses a layered rollout matrix.
   - Unit tests
   - App/service tests
   - A small set of CLI integration tests
   - A full `go test ./...` regression pass

4. Legacy repos without manifests are verified and documented only.
   - No narrow compatibility bridge is added in Phase 5.

## Notes

- An exhaustive precedence matrix was explicitly deferred to future work.
- The phase should avoid reopening the manifest-backed-only operational rule.
