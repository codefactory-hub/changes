# AGENTS.md

## Go toolchain paths for agents

Prefer plain Go commands first in this repository:

```bash
go test ./...
```

Only add writable-path overrides when the environment actually requires them, such as a sandbox or permission error involving Go's cache or module workspace.

When overrides are needed, keep them minimal:

- Start with `GOCACHE` only.
- Use `/tmp` or a repo-local cache directory for writable cache data.
- Add `GOMODCACHE`, `GOPATH`, or `GOBIN` only if the specific command fails without them.
- Do not redirect Go paths preemptively when the plain command already works.

Example fallback:

```bash
env GOCACHE=/tmp/changes-go-build go test ./...
```

If a stricter sandbox still requires repo-local writable paths, use a repo-local fallback intentionally rather than by default.
