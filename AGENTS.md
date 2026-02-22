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

## Starting Work

- Before making any changes, run `git status` and `git diff` to identify pre-existing uncommitted work in the worktree.
- Do not stage or commit files you did not modify. Pre-existing dirty files belong to other sessions or the user.
- If the worktree is dirty, note which files are already modified/untracked so you can exclude them when committing.

## Workspace Isolation (Required)

- Treat each milestone as an isolated unit of work in its own git worktree and branch.
- Before editing milestone code, create a dedicated worktree and branch from `origin/main`:
  - `make milestone-worktree NUM=<NN> SLUG=<slug>`  (from the main repo worktree)
  - This fetches origin/main and creates the worktree at `../shiru-m<NN>` on branch `milestone/<NN>-<slug>`.
  - **Never** branch from a previous milestone branch — always from `origin/main`.
- Perform all edits, tests, and commits for that milestone only inside that worktree.
- Never mix files from different milestones in one branch/worktree.
- Open a separate PR per milestone branch.
- Before starting the next milestone, return to the main repo and create a new worktree.
- If the current branch/worktree name does not match the milestone being implemented, stop and fix setup first.

### Mandatory Pre-Edit Checklist

1. Confirm the milestone ID.
2. Confirm the branch name includes the milestone ID.
3. Confirm the worktree path includes the milestone ID.
4. Run `git status --short` and ensure only milestone-related files are touched.

## Milestone Loop (Required)

- Each milestone implementation must use the milestone automation loop.
- Run milestone work through implement + adversarial review + fix rounds with:
  - `make milestone-loop MILESTONE=<MILESTONE_ID> GOAL="<implementation goal>"`
- Set `MAX_ROUNDS`, `AMP_MODE`, and `AMP_VISIBILITY` as needed for the milestone.
- Do not mark a milestone complete until the loop exits with `Final status: APPROVED`.
- If the loop exits rejected, continue fixing in the same milestone worktree and rerun the loop.

## Committing Changes

- Separate each **logical change** into its own commit. A bug fix, a refactor, and a new feature should be three separate commits, even if they touch the same file.
- Conversely, if a single logical change spans multiple files, group it into one commit.
- Each commit must build and pass tests on its own — never break the tree mid-series.
- If one commit depends on another, note it in the commit message.
- Write commit messages in imperative mood (e.g. "Add endpoint for X", not "Added endpoint for X").
- Do not mix code moves/renames with behavioral changes in the same commit. Separate the mechanical move from the functional change.

## Pre-push Checks

- A pre-commit hook in `scripts/hooks/pre-commit` runs build, test, lint, semgrep, and deadcode automatically.
- Install via `./scripts/install-tools.sh` which also sets `core.hooksPath`.
- Bypass with `git commit --no-verify` when needed (e.g. WIP commits).

### Common semgrep rules to watch for
- **`math-random-used`**: use `crypto/rand` instead of `math/rand` — even for non-security randomness.
- **`use-tls`**: suppress with `// nosemgrep: go.lang.security.audit.net.use-tls.use-tls` where TLS is handled externally.

## CI

- GitHub Actions: `golangci-lint` + `go test -timeout 5m ./...` + `semgrep` + `deadcode`
- All pushes and PRs are tested; markdown-only changes are skipped
- **Semgrep**: static analysis for security and correctness issues (`--error` flag = blocking)
- **Deadcode**: `golang.org/x/tools/cmd/deadcode` to detect unused code; remove dead code rather than leaving it
