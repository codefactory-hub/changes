# Coding Conventions

**Analysis Date:** 2026-04-06

## Naming Patterns

**Files:**
- Use lowercase package directories and lowercase Go file names with underscores only when they clarify purpose, such as `internal/cli/app.go`, `internal/cli/app_integration_test.go`, `internal/render/templates_builtin.go`, and `internal/templates/defaults_test.go`.
- Keep package tests co-located with the package they exercise using `_test.go`, such as `internal/config/config_test.go` and `internal/releases/releases_test.go`.

**Functions:**
- Exported API uses PascalCase for package entry points and value methods, such as `Initialize` in `internal/app/init.go`, `Status` in `internal/app/app.go`, `Load` in `internal/config/config.go`, and `Detect` in `internal/reporoot/reporoot.go`.
- Internal helpers use lowerCamelCase, such as `parseCreateOptions` in `internal/cli/create.go`, `renderRecommendationExplanation` in `internal/cli/app.go`, and `checkContext` in `internal/app/app.go`.
- Tests use descriptive sentence-style names without underscores, such as `TestReleaseRequiresExplicitBumpWhenNoImpactIsInferred` in `internal/cli/app_integration_test.go`.

**Variables:**
- Request/response data uses explicit noun names like `req`, `result`, `cfg`, `records`, `pending`, and `plan` in `internal/app/app.go`.
- Short-lived error variables stay as `err`; wrapped errors add operation context instead of renaming the variable, for example in `internal/fragments/fragments.go` and `internal/config/config.go`.
- CLI flag targets use plain names matching the flag surface, such as `currentVersion`, `recordPath`, `outputPath`, `accept`, and `override` in `internal/cli/app.go`.

**Types:**
- Request/result structs use PascalCase plus role suffixes, such as `InitializeRequest`, `StatusResult`, `ReleasePlan`, and `RenderRequest` in `internal/app/app.go`.
- Configuration and domain structs are singular nouns, such as `Config`, `RenderProfile`, `Fragment`, `ReleaseRecord`, and `ReleaseBundle` in `internal/config/config.go`, `internal/fragments/fragments.go`, and `internal/releases/*.go`.
- Unexported CLI-only types stay package-private, such as `createOptions` in `internal/cli/create.go` and `stringSliceFlag` in `internal/cli/stringslice.go`.

## Code Style

**Formatting:**
- Use standard Go formatting. No repository-local formatter config such as `.editorconfig`, `gofmt` wrapper config, or `goimports` config is present at the repo root.
- Keep import blocks grouped by standard library first, then a blank line, then module/external imports. See `internal/cli/app.go`, `internal/app/init.go`, and `internal/render/render_test.go`.
- Prefer explicit struct literals and explicit field names for non-trivial data setup, as in `internal/app/app_test.go` and `internal/render/render_test.go`.

**Linting:**
- No dedicated linter config such as `.golangci.yml` or `staticcheck` settings is detected in the repository root.
- CI quality enforcement is currently test-only: `.github/workflows/ci.yml` runs `go test ./...` and nothing else.

## Import Organization

**Order:**
1. Go standard library imports.
2. Third-party packages when needed, currently mostly `github.com/BurntSushi/toml` and `github.com/Masterminds/semver/v3`.
3. Internal module packages under `github.com/example/changes/internal/...`.

**Path Aliases:**
- No path alias system is used.
- Import aliases are rare and purposeful. Use aliases only to disambiguate layers, such as `appsvc` in `internal/cli/app.go`.

## Error Handling

**Patterns:**
- Return errors instead of logging or panicking in production code. The only process exit is in `cmd/changes/main.go`, after `App.Run` returns an error.
- Wrap lower-level failures with operation context using `fmt.Errorf("context: %w", err)`, as in `internal/config/config.go`, `internal/fragments/fragments.go`, `internal/releases/releases.go`, and `internal/app/init.go`.
- Validate flags and inputs early with direct, user-facing error strings, for example `release: --accept and --override cannot be combined` in `internal/cli/app.go` and `create: body is required...` in `internal/cli/create.go`.
- Use sentinel errors only when callers need stable classification. `internal/reporoot/reporoot.go` exposes `ErrNotGitRepo`, and tests assert it with `errors.Is` in `internal/reporoot/reporoot_test.go`.
- Check `context.Context` explicitly at service boundaries instead of threading it through every helper. `internal/app/app.go` and `internal/app/init.go` call `checkContext(ctx)` at multiple boundary points.

## Logging

**Framework:** None

**Patterns:**
- Do not use `log`, `slog`, or structured logging for normal execution. No logging framework is imported in repository code.
- CLI output is written directly to `App.Stdout` and `App.Stderr` with `fmt.Fprint*` in `internal/cli/app.go`.
- Error reporting is centralized through `(*App).fail` in `internal/cli/app.go`, which prefixes stderr output with `error:`.

## Comments

**When to Comment:**
- Keep comments sparse. Most files rely on descriptive names rather than line comments.
- Use comments only where a behavioral exception needs explanation, such as the empty-mode allowance in `internal/config/config.go` and scaffold guidance comments in `internal/cli/create.go`.

**JSDoc/TSDoc:**
- Not applicable. This is a Go codebase.
- Go doc comments on exported identifiers are generally not present in `internal/*` packages. Match current style unless the package becomes externally consumed.

## Function Design

**Size:** Large coordinator functions are acceptable at layer boundaries.
- `(*App).Run` and subcommand handlers in `internal/cli/app.go` centralize CLI parsing and output.
- `Initialize` flow in `internal/app/init.go` and release flow in `internal/app/app.go` coordinate multiple package calls, while deeper helpers stay focused.

**Parameters:** Prefer explicit parameter objects or injected collaborators over long positional signatures.
- Use request/result structs for application services, such as `StatusRequest` and `RenderResult` in `internal/app/app.go`.
- Inject environment-sensitive behavior through fields or helper structs, such as `App.Now`, `App.Random`, `App.IsTTY`, `App.EditFile` in `internal/cli/app.go`, and `initializeDeps` in `internal/app/init.go`.

**Return Values:** Return concrete values plus `error`.
- Constructors and loaders usually return a typed value and `error`, such as `Load` in `internal/config/config.go` and `Create` in `internal/fragments/fragments.go`.
- Mutating operations often return a result object with the persisted path or record, such as `CommitReleaseResult` in `internal/app/app.go`.

## Module Design

**Exports:** Export package-level operations, keep helpers private.
- `internal/app`, `internal/config`, `internal/fragments`, `internal/releases`, and `internal/render` expose the package API.
- Helper functions like `normalizeType`, `selectReleaseFragments`, and `buildCreateScaffold` stay unexported inside their packages.

**Barrel Files:** Not used.
- There are no re-export or index-style files. Import the concrete package directly, such as `github.com/example/changes/internal/config` or `github.com/example/changes/internal/releases`.

## CLI Patterns

- Parse each subcommand with its own `flag.FlagSet` configured with `flag.ContinueOnError` and `fs.SetOutput(io.Discard)`, as in `internal/cli/app.go` and `internal/cli/create.go`.
- Treat help as normal output, not an error path. `isHelpArg`, `wantsHelp`, and `printHelp` in `internal/cli/app.go` short-circuit before command execution.
- Write success and informational text to stdout, and reserve stderr for failures and interactive prompts. See `internal/cli/app.go` and `internal/cli/create.go`.
- Keep repo discovery in one place through `(*App).repoRoot` in `internal/cli/app.go` instead of resolving paths inside each command.

## Example Patterns

**Import grouping and explicit dependency aliasing:**
```go
import (
	"context"
	"flag"
	"fmt"

	appsvc "github.com/example/changes/internal/app"
	"github.com/example/changes/internal/config"
)
```

**Wrapped errors with operation context:**
```go
if err := os.MkdirAll(dir, 0o755); err != nil {
	return Fragment{}, fmt.Errorf("create fragments directory: %w", err)
}
```

**Dependency injection for environment-sensitive behavior:**
```go
app := &App{
	Stdout: stdout,
	Stderr: stderr,
	Stdin:  os.Stdin,
	Now:    time.Now,
}
```

---

*Convention analysis: 2026-04-06*
