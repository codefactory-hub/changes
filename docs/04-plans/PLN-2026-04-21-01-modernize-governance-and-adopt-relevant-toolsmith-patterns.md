---
created: 2026-04-21
governance_audit: remediation-plan
---

# PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns

## Why now

`changes` already has real local ADR history and a stable CLI identity, but its governance layout is still in the legacy `docs/decisions/`, `docs/plans/`, and `docs/research/` form. Its live policy surfaces also do not yet describe the current canonical lifecycle or the subset of `toolsmith` tool-pattern ADRs that fit this repository's actual boundaries.

This remediation needs to preserve local ADR history, migrate documentation conservatively, and avoid importing irrelevant auth or secret-handling patterns into a repo whose CLI core does not currently read dotenv files or provider-specific secret references.

## Audit summary

- The worktree is clean, so implementation may proceed under the repo's clean-worktree policy.
- Local numbered ADRs `0001` through `0006` remain valid governing decisions for this repository and should be preserved as local ADRs.
- The repo currently lacks explicit local governance in the canonical lifecycle model.
- Historical governed docs live under legacy folder names and should be migrated without renumbering or silently rewriting their intent.
- The only secret-bearing surface found in-repo is release automation and local release-helper execution through environment variables such as `CHANGES_HOMEBREW_TAP_TOKEN`.
- The CLI core does not directly read dotenv files and does not expose provider-specific secret references or generic authenticated shell behavior.

## Pattern-adoption decisions

Adopt as local numbered ADRs:

- canonical repository-governance lifecycle for `changes` as a receiving tool repo
- separation between human local release auth paths and agent or automation auth paths
- provider-neutral secret delivery for the release-helper secret surface where it actually exists in this repo

Do not adopt now:

- checked-in nonsecret 1Password edge-mapping policy, because this repo does not currently keep repo-tracked launcher profiles or `.env.1p`-style artifacts
- resolved-env-file and pre-launch-resolution policy as a numbered ADR, because the tool core and release helpers do not directly load env files today

The implementation should still keep unresolved provider references out of the tool core and describe external launcher resolution at the edge when documentation needs to mention it.

## Planned work

1. Create the canonical governance folder skeleton under `docs/` and move the legacy governed docs into their canonical lifecycle folders without renumbering existing local ADR files.
2. Add new local ADRs after `ADR-0006` to establish canonical governance and the relevant release-auth policy for this repository.
3. Update live policy surfaces in `README.md` and `AGENTS.md` so they describe the canonical lifecycle, planning discipline, current ADR set, and the treatment of pre-baseline historical records.
4. Update release documentation and helper scripts so the repo's actual secret-bearing release surface matches the adopted local policy.
5. Validate the resulting docs and implementation, then use the `git-workflow` skill as prudent to finish with committed changes and a clean worktree.

## Validation

- Run `go test ./...`.
- Run `bash -n` on the release-helper scripts.
- Check for stale references to legacy governed-doc paths in live policy surfaces.
- Confirm the final worktree is clean after committing the completed work.

## Git discipline

Use the `git-workflow` skill as often as prudent to create semantically cohesive commits during this implementation. Run Git actions in series only. The last implementation step is to commit the completed work and leave the working directory clean.
