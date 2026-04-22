---
created: 2026-04-21
status: accepted
---

# ADR-0009: Adopt provider-neutral release secret ingestion

## Context

The `changes` CLI core does not have a broad auth surface, but the repository's release helpers and publishing flow do depend on secret-bearing inputs such as `CHANGES_HOMEBREW_TAP_TOKEN`.

Those inputs should remain stable across GitHub Actions secrets, local shell exports, file-backed secret delivery, and external launchers such as 1Password or other secret managers. The repository needs a durable local rule for that narrow secret-bearing surface without pretending that `changes` is a general-purpose authenticated CLI.

## Decision

- Release-helper secret inputs use logical environment names. The repository does not require provider-specific references as runtime arguments.
- `CHANGES_HOMEBREW_TAP_TOKEN` is accepted through either `CHANGES_HOMEBREW_TAP_TOKEN` or `CHANGES_HOMEBREW_TAP_TOKEN_FILE`.
- If both `CHANGES_HOMEBREW_TAP_TOKEN` and `CHANGES_HOMEBREW_TAP_TOKEN_FILE` are set, release helpers fail fast with a clear error.
- Release helpers do not accept raw secret values on argv.
- Repo-tracked docs describe the logical contract shape and leave provider-specific resolution to launcher, wrapper, or runner configuration outside the tool core.
- Unresolved provider references belong at the launch edge, not in `changes` core code or in release-helper arguments.

## Consequences

The release flow keeps one stable secret contract while remaining compatible with local shell exports, file-backed secret delivery, GitHub Actions secrets, and external secret launchers. The repository does not need checked-in provider mappings or direct dotenv loading to support local release work.
