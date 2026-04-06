#!/usr/bin/env bash
set -euo pipefail

tag_name="${1:-${GITHUB_REF_NAME:-changes}}"

mkdir -p .dist

write_placeholder() {
  printf '# %s\n\nRelease notes placeholder. Commit repo-local `changes` config and templates and create a first final release record before enabling generated release notes.\n' "${tag_name}" > .dist/release-notes.md
}

if [[ ! -f .config/changes/config.toml ]]; then
  write_placeholder
  exit 0
fi

if ! go run ./cmd/changes render --latest --profile repository_markdown --output .dist/release-notes.md 2>/dev/null; then
  write_placeholder
  printf 'prepare-release-notes: using placeholder release notes\n' >&2
fi
