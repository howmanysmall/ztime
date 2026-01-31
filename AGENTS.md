# AGENTS

## Purpose

This file guides coding agents working in this repo.
Follow commands and style rules below.

## Repo facts

- Module: github.com/howmanysmall/ztime (Go 1.25.6)
- Entry point: src/main.go (binary name: ztime)
- CI: .github/workflows/ci.yaml (build, test, lint)
- Lint config: .golangci.json (gofumpt + many linters)
- Formatting configs: biome.jsonc, tombi.toml, .markdownlint.json
- Cursor/Copilot rules: none found in .cursor/rules, .cursorrules, .github/copilot-instructions.md

## Core commands

Use Go 1.25.x when possible (CI uses 1.25).

### Build

```bash
mkdir -p /tmp/ztime-build
go build -v -o /tmp/ztime-build ./...
go build -o ztime ./src
```

### Test

```bash
go test -v -race ./...
go test -v -race -coverprofile=coverage.txt ./...
```

### Lint and type-check

```bash
golangci-lint run ./...
go vet ./...
```

### Format

```bash
go fmt ./...
gofumpt -w ./src
```

### Single test patterns

```bash
go test ./src -run '^TestName$' -count=1
go test ./src -run '^TestName$/^Subtest$' -count=1
go test ./... -run '^TestName$' -count=1
```

### Package.json scripts (bun or npm)

```bash
bun run format
bun run lint
bun run test
bun run type-check
npm run format
npm run lint
npm run test
npm run type-check
```

### Release housekeeping

- GoReleaser runs `go mod tidy` before builds (see .goreleaser.yaml).
- If modules change, run `go mod tidy` and commit the result.

## Go code style

### Formatting and layout

- Use gofumpt formatting (gopls is configured with gofumpt).
- Tabs for indentation; keep gofmt/gofumpt output clean.
- Keep lines readable; avoid deeply nested blocks (nestif, gocognit).
- No trailing whitespace.

### Imports

- Group imports: standard library, blank line, third-party, blank line, local.
- Avoid dot imports and unnamed imports unless needed for side effects.
- Use import aliases only to resolve collisions or clarify (importas).
- Remove unused imports; do not rely on organizeImports in editor.

### Naming

- Exported identifiers: PascalCase; unexported: lowerCamel.
- Receiver names: short and consistent (e.g., `s`, `c`, `b`), no underscores.
- Error variables: `ErrX` for exported, `errX` for internal.
- Avoid stutter in exported names (Go convention).

### Types and APIs

- Prefer concrete types; accept interfaces when required by callers.
- Thread `context.Context` through call chains when doing I/O.
- Avoid globals and `init()` (gochecknoglobals, gochecknoinits).
- Remove unused params and return values (unparam, unused).
- Keep functions short (funlen) and complexity low (gocognit).
- Prefer range loops when index is unused (intrange).
- Preallocate slices when size is known (prealloc).
- Avoid unnecessary conversions (unconvert) and assignments (wastedassign).

### Error handling

- Check every error return (errcheck).
- Wrap errors with `%w` and add context (`fmt.Errorf("...: %w", err)`).
- Use `errors.Is`/`errors.As` for comparisons (errorlint).
- Prefer sentinel errors for comparisons (err113).
- Do not return `nil, nil` from functions returning `(T, error)` (nilnil).
- Avoid `panic` except for truly unrecoverable states in `main`.
- Close resources (`defer resp.Body.Close()`) after error checks (bodyclose).

### Context usage

- Use `context.Background()` only at top-level entry points.
- Do not pass `context.TODO()` or `context.Background()` when a parent context exists (noctx).

### Concurrency and loops

- When launching goroutines in loops, copy loop variables (copyloopvar).
- Keep goroutines cancellable; use contexts or channels.
- Avoid capturing large loop variables by reference.

### Testing

- Use `package <name>_test` for external tests (testpackage).
- Call `t.Helper()` in test helpers (thelper).
- Use `t.Parallel()` for safe tests and subtests (paralleltest).
- Prefer table-driven tests and clear test names.
- Avoid sleeping for timing; prefer deterministic checks.

### Security and correctness

- Validate inputs passed to `exec.Command` and file operations (gosec).
- Avoid shadowing predeclared identifiers (predeclared).
- Do not ignore `Close`/`Flush` errors in defers (errcheck).

## Non-Go formatting

### JSON, JSONC, and workspace files

- Biome formats JSON/JSONC with tabs, width 120, double quotes.
- Prefer trailing commas only where format rules allow (see biome.jsonc).

### JS/TS

- Biome formats JS/TS with tabs and semicolons.
- Keep filenames in kebab-case when adding new JS/TS files.

### TOML

- Tombi formats TOML with tabs, width 120, double quotes.

### Markdown

- Markdownlint rules allow tabs and ignore line-length checks.
- Use standard markdown headings and lists.

## When unsure

- Prefer existing patterns in `src/main.go` and CI config.
- Check `.golangci.json` for lint-driven constraints.
- Ask for clarification only if a choice changes behavior or UX.

## Suggested workflow

- Make changes.
- Run `golangci-lint run ./...` and `go test -v -race ./...`.
- If docs/config changed, run the matching formatter.
