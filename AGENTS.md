# Shiru — Go Backend Service

## Build & Test

- **Go version**: 1.23+ (see `go.mod`)
- **Run tests**: `go test -timeout 5m ./...`
- **Run single test**: `go test -run TestName ./path/to/package`
- **Lint**: `golangci-lint run`
- **Generate mocks**: `go generate ./...`

## Project Structure

- `main.go` at root (single binary entrypoint)
- `internal/` for all private packages, organized by domain (e.g. `internal/config/`, `internal/<domain>/`)
- Mocks live in a `mock/` subdirectory of the package they mock

## Code Conventions

### Style
- Follow `gofmt` and `goimports` formatting
- No comments on exported symbols unless they add real value beyond the name
- Constants grouped at the top of the file
- Use `snake_case` for JSON struct tags

### Error Handling
- Return errors up the call stack; do not swallow them
- Wrap errors with `fmt.Errorf("context: %w", err)` to preserve the chain
- Use sentinel errors (e.g. `var ErrNotFound = ...`) and check with `errors.Is`
- Use `go.uber.org/multierr` when accumulating multiple errors

### Logging
- Use `go-logr/logr` for structured logging with key-value pairs: `log.Info("message", "key", value)`
- Log errors with `log.Error(err, "message", "key", value)`

### Interfaces & Mocking
- Define interfaces next to their primary implementation
- Use `var _ Interface = &Impl{}` for compile-time interface satisfaction checks
- Generate mocks with `go.uber.org/mock/mockgen` via `//go:generate` directives
- Place generated mocks in `mock/` subdirectory: `//go:generate go run go.uber.org/mock/mockgen -destination mock/<file>.go -package mock . <Interface>`

### Configuration
- Use `github.com/sethvargo/go-envconfig` with struct tags for environment-based config
- Config structs live in `internal/config/`

### Constructors
- `func New(...) *Type` for infallible construction
- `func NewFoo(...) (*Foo, error)` when initialization can fail

## Testing Conventions

- Use `github.com/stretchr/testify/require` for assertions that should stop the test on failure
- Use `github.com/stretchr/testify/assert` for non-fatal assertions
- Write table-driven tests with `t.Run` and `t.Parallel()`
- Tests live in the same package (internal tests), not `_test` packages
- Extract reusable test helpers (e.g. `initialState()`, `notFound()`, `noErr()`)
- Use `go.uber.org/mock/gomock` for mock expectations in tests

## Committing Changes

- Separate each **logical change** into its own commit. A bug fix, a refactor, and a new feature should be three separate commits, even if they touch the same file.
- Conversely, if a single logical change spans multiple files, group it into one commit.
- Each commit must build and pass tests on its own — never break the tree mid-series.
- If one commit depends on another, note it in the commit message.
- Write commit messages in imperative mood (e.g. "Add endpoint for X", not "Added endpoint for X").
- Do not mix code moves/renames with behavioral changes in the same commit. Separate the mechanical move from the functional change.

## Pre-push Checks

Run these locally **before pushing** to catch CI failures early:

```bash
go build ./...                        # must compile
go test -timeout 5m ./...             # all tests must pass
golangci-lint run --timeout=5m        # lint must pass (or: go tool golangci-lint run --timeout=5m)
semgrep scan --config auto --error    # semgrep must report 0 blocking findings
deadcode ./...                        # review output; remove any dead code you own
```

### Common semgrep rules to watch for
- **`math-random-used`**: use `crypto/rand` instead of `math/rand` — even for non-security randomness.
- **`use-tls`**: suppress with `// nosemgrep: go.lang.security.audit.net.use-tls.use-tls` where TLS is handled externally.

## CI

- GitHub Actions: `golangci-lint` + `go test -timeout 5m ./...` + `semgrep` + `deadcode`
- All pushes and PRs are tested; markdown-only changes are skipped
- **Semgrep**: static analysis for security and correctness issues (`--error` flag = blocking)
- **Deadcode**: `golang.org/x/tools/cmd/deadcode` to detect unused code; remove dead code rather than leaving it
