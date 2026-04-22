#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib/secret_env.sh
source "${script_dir}/lib/secret_env.sh"

export GOCACHE="${PWD}/.cache/go-build"
export GOPATH="${PWD}/.local/share/go"
export GOMODCACHE="${PWD}/.local/share/go/pkg/mod"
export GOBIN="${PWD}/.local/share/go/bin"

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

goreleaser build --snapshot --clean
