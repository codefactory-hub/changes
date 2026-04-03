# AGENTS.md

## Go toolchain paths for agents

When running Go commands inside this repository from Codex or other sandboxed agents, keep Go's writable cache and workspace data inside the repository so the toolchain does not try to write into user-level directories outside the sandbox.

Use this repo-local XDG-style layout:

- `GOCACHE=$PWD/.cache/go-build`
- `GOPATH=$PWD/.local/share/go`
- `GOMODCACHE=$PWD/.local/share/go/pkg/mod`
- `GOBIN=$PWD/.local/share/go/bin`

Apply those variables to commands that may write Go cache or module data, including `go test`, `go build`, `go run`, and `go install`.

Example:

```bash
env GOCACHE=$PWD/.cache/go-build \
  GOPATH=$PWD/.local/share/go \
  GOMODCACHE=$PWD/.local/share/go/pkg/mod \
  GOBIN=$PWD/.local/share/go/bin \
  go test ./...
```

These paths are agent/tooling-only. They are separate from the committed `.local/share/changes/**` source-of-truth data used by the `changes` CLI itself.
