# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`wt` is a Go CLI that wraps `git worktree` to:
1. Create a branch (if it doesn't exist) from a base branch
2. Create a git worktree for that branch at a derived path
3. Copy gitignored files listed in `.worktreeinclude` into the new worktree (full gitignore spec including negation)

The bash prototype (`wt`) is kept for reference only. The Go implementation lives under `cmd/` and `internal/`.

## Building and running

```bash
# Build
go build -o wt-bin ./cmd/wt

# Run locally without installing
./wt-bin <branch> [options]

# Dry-run to preview behavior without side effects
./wt-bin feature/myBranch --dry-run

# Help / version
./wt-bin --help
./wt-bin --version
```

## Testing

```bash
# Unit tests (all packages)
go test ./...

# Integration tests (require git in PATH, compile the binary internally)
go test -tags=integration ./tests/

# Verbose output
go test -v ./internal/worktree/
```

## Project structure

```
cmd/wt/main.go               # Entry point — injects version via ldflags
internal/
  cli/
    root.go                  # Cobra command, flag parsing, run() orchestration
    output.go                # info/warn/die helpers with ANSI TTY detection
    output_test.go
  naming/
    derive.go                # Derive(branch) string — branch → kebab-case worktree name
    validate.go              # ValidateWorktreeName(name) error
    derive_test.go
    validate_test.go
  gitx/
    git.go                   # Git interface
    real.go                  # realGit — executes git via os/exec
    fake.go                  # FakeGit — in-memory stub for unit tests
    fake_test.go
    real_test.go             # build tag: integration
  fsx/
    fs.go                    # FS interface
    real.go                  # realFS — wraps os package
    afero.go                 # aferoFS — wraps afero.MemMapFs for unit tests
    real_test.go
    afero_test.go
  worktree/
    plan.go                  # Plan and Options structs
    include.go               # .worktreeinclude parser + git-ignore matcher
    include_test.go
    service.go               # Service: BuildPlan() + Execute()
    service_test.go
tests/
  integration_test.go        # End-to-end tests (build tag: integration)
.github/workflows/
  ci.yml                     # Lint + test on PR (matrix: ubuntu/macos/windows)
  release.yml                # GoReleaser on tag v*
.goreleaser.yaml             # Cross-compile + Homebrew tap + Scoop bucket
install.sh                   # Curl-installer: detects OS/arch, verifies SHA256
```

## Key logic

### Branch-to-worktree name derivation (`internal/naming/derive.go`)

`/` → `-`, camelCase → kebab-case, then lowercase. Pure function, fully tested.

```
feature/loginForm    → feature-login-form
bugfix/fixApiTimeout → bugfix-fix-api-timeout
hotfix-urgent        → hotfix-urgent
```

### `.worktreeinclude` (`internal/worktree/include.go`)

Reads patterns from `.worktreeinclude` (gitignore syntax), calls `git ls-files --others --ignored --exclude-standard` with the positive patterns as scope, then applies the full pattern set via `go-gitignore` — which supports negation (`!pattern`). This is the key fix over the bash version.

### Validation

- Branch name: `git check-ref-format --branch` via `gitx.Git.CheckRefFormat`
- Worktree name: rejects empty, `..`, absolute paths, names with spaces
- Destination: must not already exist before `git worktree add`

### Interfaces for testability

`gitx.Git` and `fsx.FS` are injected into `worktree.Service`. Unit tests use `gitx.FakeGit` (in-memory) and `fsx.NewAfero(afero.NewMemMapFs())`. Integration tests compile the real binary and run it against a temp git repo.

## Style constraints

- No comments unless the WHY is non-obvious
- Errors wrapped with `fmt.Errorf("context: %w", err)` — never swallowed
- `SilenceUsage: true` on the cobra command (errors don't re-print usage)
- Output always via `cli.Output` helpers (`Info`, `Warn`, `Die`) — never `fmt.Print` directly in `RunE`
- All git side effects gated behind `opts.DryRun` check in `run()`
