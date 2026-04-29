---
created: 2026-04-29
---

# PLN-2026-04-29-01: Add JSON output and ignore hygiene

## Why now

The CLI review found three concrete gaps: data-producing commands that require parsing human output, a `render profiles` subcommand that ignores unexpected positional arguments, and a repo-local `.gitignore` that omits current Go and macOS baseline entries.

## Planned work

1. Add `--json` support for `changes status` and `changes render profiles` while preserving existing human output.
2. Reject unexpected positional arguments for `changes render profiles`.
3. Expand `.gitignore` with the relevant `gibo dump Go macOS` baseline entries while keeping the repository's existing repo-local cache and state ignores.
4. Add focused CLI integration coverage for the new JSON surfaces and argument validation.
5. Use the `git-workflow` skill as prudent to finish committing the work and leave the worktree clean.

## Validation

- Run `go test ./...`.
- Confirm the final worktree is clean after the implementation commit.
