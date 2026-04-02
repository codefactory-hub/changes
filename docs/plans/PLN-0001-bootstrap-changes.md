# PLN-0001 Bootstrap Changes

## Scope

Ship the first sound layer of `changes` as a Go CLI that can operate inside any Git repository with repo-local configuration, durable fragments, release manifests, deterministic version suggestion, template-based rendering, and changelog rebuilds.

The first layer includes:

- `changes init`
- `changes add`
- `changes status`
- `changes version next`
- `changes release create`
- `changes render`
- `changes changelog rebuild`
- default repo-local templates
- tests for fragment naming, versioning, manifest consumption semantics, render truncation, and changelog determinism
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
- `.local/state/changes` is transient and ignored.
- Fragments are durable artifacts and are not deleted when released.
- Release manifests freeze fragment selection at release creation time.
- Preview releases are line-local deltas and do not consume fragments globally.
- Stable releases consume fragments logically through the manifest graph.
- Rendering must enforce `max_chars` by dropping complete entries from the bottom only.

## Follow-up work

- editor integration for fragment body authoring
- richer filters and custom metadata
- explicit manifest selection controls beyond the default unreleased set
- changelog section customization beyond template overrides
- repository self-hosting of `changes` for its own release notes
- private tap automation hardening once org-specific endpoints and tokens are known
