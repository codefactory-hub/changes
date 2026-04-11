# Changelog

## 0.1.2 (stable)

### Fixed

- Omit unset integer ordering fields from fragment front matter. `changes create` no longer serializes `release_notes_priority = 0` or `display_order = 0` when those flags were not provided on the command line.

## 0.1.1 (stable)

### Fixed

- Narrow the repo-local xdg .gitignore rule to the authoritative state directory. Repo-local xdg init and repair now ignore /.local/state/changes/ instead of the broader /.local/state/ parent directory, so unrelated repo-local state paths are not hidden.

## 0.1.0 (stable)

### Added

- Add release rendering through named render profiles and built-in templates. This includes repository changelog rendering, GitHub/GitLab release-note rendering, tester summaries, and package-manager-oriented output formats derived from canonical release history.
- Add adoption bootstrap support for repositories that start using `changes` after they already have released versions. `changes init --current-version <semver|unreleased>` can now establish a release-history baseline, create a standard adoption release when needed, and generate a repo-specific LLM prompt for reconstructing older release history.
- Standardize release automation around GoReleaser, GitHub Actions, and a Homebrew tap workflow. The repository now includes local release-preparation scripts, tag-driven GitHub release automation, and documentation for publishing binaries and Homebrew metadata from the recorded release history.
- Introduce the core `changes` release model for Git repositories. This adds repo-local configuration, durable fragment storage, canonical release records, and explicit `status` / `release` workflows for assembling and recording releases from pending fragments.
- Derive release impact from semantic fragment levers instead of storing an explicit bump in each fragment. Fragments can now describe `public_api`, `behavior`, `dependency`, and `runtime`, and `changes` combines those facts with the repository's public-API stability policy to recommend the next version.

### Changed

- Rename the fragment authoring command to `create` and add guided authoring. Prompt for type, optional name stem, and body text in TTY sessions, add `--edit` for scaffolded editor-driven drafting, and include name stems in generated fragment filenames.
- Add flexible global and repo-local layout support for `xdg` and `home`. `changes` can now resolve configuration, data, and state through either XDG-style directories or a single-root home layout, inspect authority with `changes doctor`, and fail loudly when multiple supported layouts compete.
