#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/secret_env.sh
source "${script_dir}/lib/secret_env.sh"

require_goreleaser() {
  if ! command -v goreleaser >/dev/null 2>&1; then
    echo "error: goreleaser is not installed" >&2
    exit 1
  fi
}

goreleaser_version() {
  goreleaser --version | awk '/GitVersion:/ { print $2; exit }'
}

version_is_supported() {
  local raw="${1#v}"
  local major minor
  IFS='.' read -r major minor _ <<<"${raw}"
  [[ -n "${major:-}" && -n "${minor:-}" ]] || return 1
  if (( major > 2 )); then
    return 0
  fi
  if (( major == 2 && minor >= 10 )); then
    return 0
  fi
  return 1
}

require_goreleaser

version="$(goreleaser_version)"
if ! version_is_supported "${version}"; then
  echo "error: this repository requires GoReleaser v2.10 or newer; found ${version}" >&2
  echo "hint: .goreleaser.yaml uses homebrew_casks, which is supported starting in v2.10" >&2
  exit 1
fi

export CHANGES_GITHUB_OWNER="${CHANGES_GITHUB_OWNER:-example-owner}"
export CHANGES_GITHUB_REPO="${CHANGES_GITHUB_REPO:-changes}"
export CHANGES_PROJECT_HOMEPAGE="${CHANGES_PROJECT_HOMEPAGE:-https://example.invalid/changes}"
export CHANGES_HOMEBREW_TAP_OWNER="${CHANGES_HOMEBREW_TAP_OWNER:-example-owner}"
export CHANGES_HOMEBREW_TAP_REPO="${CHANGES_HOMEBREW_TAP_REPO:-homebrew-tap}"
export CHANGES_HOMEBREW_TAP_BRANCH="${CHANGES_HOMEBREW_TAP_BRANCH:-main}"
resolve_secret_input CHANGES_HOMEBREW_TAP_TOKEN
export CHANGES_HOMEBREW_TAP_TOKEN="${CHANGES_HOMEBREW_TAP_TOKEN:-placeholder-token}"
export CHANGES_RELEASE_BOT_NAME="${CHANGES_RELEASE_BOT_NAME:-Changes Release Bot}"
export CHANGES_RELEASE_BOT_EMAIL="${CHANGES_RELEASE_BOT_EMAIL:-changes@example.invalid}"

goreleaser check
