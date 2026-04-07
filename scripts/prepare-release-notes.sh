#!/usr/bin/env bash
set -euo pipefail

tag_name="${1:-${GITHUB_REF_NAME:-changes}}"
release_version="${tag_name#v}"
release_notes_path=".local/share/changes/releases/changes-${release_version}+github_release.md"
compat_notes_path=".dist/release-notes.md"

mkdir -p .dist
mkdir -p "$(dirname "${release_notes_path}")"

write_placeholder() {
  printf '# %s\n\nRelease notes placeholder. Commit repo-local `changes` config and templates and create a first final release record before enabling generated release notes.\n' "${tag_name}" > "${release_notes_path}"
  cp "${release_notes_path}" "${compat_notes_path}"
}

if [[ ! -f .config/changes/config.toml ]]; then
  write_placeholder
  exit 0
fi

if ! go run ./cmd/changes render --version "${release_version}" --profile github_release --output "${release_notes_path}" 2>/dev/null; then
  write_placeholder
  printf 'prepare-release-notes: using placeholder release notes\n' >&2
  exit 0
fi

cp "${release_notes_path}" "${compat_notes_path}"
