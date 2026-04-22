# PLN-0001 Bootstrap Changes

Historical planning record. The command names and scope below reflect the plan at the time it was written, not the current CLI surface.

## Scope

Ship the first sound layer of `changes` as a Go CLI that can operate inside any Git repository with repo-local configuration, durable fragments, release records, deterministic version suggestion, template-based rendering, and changelog rebuilds.

The first layer includes:

- `changes init`
- `changes add`
- `changes status`
- `changes version next`
- `changes release create`
- `changes render`
- `changes changelog rebuild`
- default repo-local templates
- tests for fragment naming, versioning, parent-linked release-record lineage, render-profile truncation, and changelog determinism
- GoReleaser and GitHub Actions bootstrap for binary delivery and private Homebrew cask publishing

## Non-goals

- monorepo multi-package orchestration
- interactive editor workflows beyond `--body`
- remote state, SaaS backends, or service coordination
- localization, signing/notarization completeness, or rich TUI work
- exhaustive semver policy for every prerelease edge case

## Design constraints

- Git root detection is mandatory; operating outside a Git repository is an error.
- Repo-local `.config` and `.local/share` paths are committed source-of-truth.
- `.local/state` is transient and ignored.
- Fragments are durable artifacts and are not deleted when released.
- Base release records freeze fragment selection at release creation time.
- Base release records are append-only, parent-linked selection records that store only the fragment IDs newly added by that release record.
- Preview releases are line-local deltas anchored to their own preview lineage.
- Stable releases remain on the stable lineage even when previews exist for the same target.
- Rendering policy lives in named render profiles, not in release records.
- Multi-release rendering must enforce `max_chars` by dropping complete release records from the bottom only.

## Follow-up work

- editor integration for fragment body authoring
- richer filters and custom metadata
- explicit release-record selection controls beyond the default unreleased set
- changelog section customization beyond template overrides
- repository self-hosting of `changes` for its own release notes
- private tap automation hardening once org-specific endpoints and tokens are known
