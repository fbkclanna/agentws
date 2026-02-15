# Development Guide

> [日本語](DEVELOPMENT.ja.md)

Technical guide for contributing to agentws.

## Prerequisites

- Go 1.26.0 or later
- Git 2.25 or later
- golangci-lint
- lefthook
- GoReleaser

## Pre-commit hook

lefthook automatically runs build, lint, and test before each commit.

```bash
# Install
go install github.com/evilmartians/lefthook@latest

# Enable (installs hook at .git/hooks/pre-commit)
lefthook install
```

## Build

```bash
# Development build
go build ./cmd/agentws

# Build with version embedding
go build -ldflags "-X main.version=1.0.0" ./cmd/agentws

# Cross-compile (GoReleaser)
goreleaser release --snapshot --clean
```

## Test

```bash
# Run all tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# Specific package
go test ./internal/manifest/...
go test ./cmd/agentws/...
```

### Writing Tests

Tests often require a Git repository. The `internal/testutil` package provides helpers.

```go
import "github.com/fbkclanna/agentws/internal/testutil"

func TestSomething(t *testing.T) {
    // Create a bare repo with an initial commit (main branch)
    bare := testutil.CreateBareRepo(t)

    // Create a bare repo with an additional branch
    bare := testutil.CreateBareRepoWithBranch(t, "feature/x")
}
```

For CLI command tests, use `newRootCmd()` to build the entire cobra command tree and pass flags via `SetArgs()`.

```go
func TestSomeCommand(t *testing.T) {
    dir := t.TempDir()
    root := newRootCmd()
    root.SetArgs([]string{"--root", dir, "sync", "--jobs", "1"})
    if err := root.Execute(); err != nil {
        t.Fatal(err)
    }
}
```

To test a command's standard output, set a `bytes.Buffer` via `cmd.SetOut()`. Always use `cmd.OutOrStdout()` for output (never `os.Stdout`).

## Lint

```bash
golangci-lint run
```

Configuration is in `.golangci.yml`. Enabled linters: errcheck, govet, staticcheck, unused, ineffassign, gosimple, gofmt, misspell.

## Project Structure

```
agentws/
├── cmd/agentws/          # CLI entry point and all subcommands
│   ├── main.go           # Entry point (version variable)
│   ├── root.go           # Root command definition & subcommand registration
│   ├── cmd_init.go       # init subcommand
│   ├── cmd_sync.go       # sync subcommand
│   ├── cmd_status.go     # status subcommand
│   ├── cmd_pin.go        # pin subcommand
│   ├── cmd_branches.go   # branches subcommand
│   ├── cmd_checkout.go   # checkout subcommand
│   ├── cmd_start.go      # start subcommand
│   ├── cmd_clean.go      # clean subcommand
│   ├── cmd_doctor.go     # doctor subcommand
│   ├── cmd_run.go        # run subcommand
│   ├── templates.go      # Template definitions for init --template
│   └── exec.go           # post_sync command execution helper
│
├── internal/
│   ├── manifest/         # workspace.yaml model and parser
│   │   ├── model.go      # Workspace, Repo, Profile, Defaults structs
│   │   └── parse.go      # YAML parsing, validation & filtering
│   │
│   ├── lock/             # workspace.lock.yaml model and parser
│   │   ├── model.go      # File, Repo structs
│   │   └── parse.go      # Load/Parse/Save
│   │
│   ├── git/              # Git command wrapper
│   │   └── git.go        # Clone, Fetch, Checkout, Branch operations, etc.
│   │
│   ├── workspace/        # Core workspace operations
│   │   └── workspace.go  # Context, Load, Strategy
│   │
│   ├── ui/               # CLI output utilities
│   │   ├── table.go      # Table-formatted output
│   │   └── progress.go   # Parallel processing progress display
│   │
│   └── testutil/         # Test helpers
│       └── repo.go       # Bare repo creation utilities
│
├── .github/workflows/ci.yml   # GitHub Actions CI
├── .golangci.yml               # Linter configuration
├── .goreleaser.yml             # Release configuration
└── workspace.yaml              # (Example manifest created by the user)
```

## Architecture

### Package Dependencies

```
cmd/agentws
  ├── internal/manifest    (Read & filter workspace.yaml)
  ├── internal/lock        (Read & write workspace.lock.yaml)
  ├── internal/git         (Execute Git operations)
  ├── internal/workspace   (Integrate manifest + lock, resolve paths)
  └── internal/ui          (Table & progress output)

internal/workspace
  ├── internal/manifest
  └── internal/lock
```

`internal/git` has no dependencies on other internal packages. `internal/manifest` and `internal/lock` are also independent of each other.

### Design Principles

1. **Defaults merging**: Per-repo settings take priority; if unspecified, values from the `defaults` section are used. Resolved via `Repo.Effective*()` methods.

2. **Strategy pattern**: Handling dirty working trees is selected from 3 options: `safe` (skip) / `stash` (stash and continue) / `reset` (force reset). `reset` requires `--force`.

3. **Filtering**: Repos are filtered through 2 layers: profile (tag/ID-based) and `--only`/`--skip` (ID-based).

4. **Parallel execution**: The sync command processes repos in parallel using goroutines + channel semaphore (controlled by `--jobs`). `ui.Progress` safely displays progress with atomic counters + mutex.

5. **Testability**: CLI output is written via `cmd.OutOrStdout()`. Tests can capture output by setting `cmd.SetOut(&buf)`.

## Adding a New Subcommand

1. Create `cmd/agentws/cmd_<name>.go`

```go
package main

import "github.com/spf13/cobra"

func newMyCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycmd",
        Short: "Description of my command",
        RunE:  runMyCmd,
    }
    // Define flags
    cmd.Flags().Bool("some-flag", false, "Flag description")
    return cmd
}

func runMyCmd(cmd *cobra.Command, args []string) error {
    root, _ := cmd.Flags().GetString("root")
    // Load workspace
    ctx, err := workspace.Load(root)
    if err != nil {
        return err
    }
    // Implement logic
    return nil
}
```

2. Register in `newRootCmd()` in `root.go`

```go
cmd.AddCommand(
    // ...existing commands...
    newMyCmd(),
)
```

3. Create tests in `cmd/agentws/cmd_<name>_test.go`

## Common Flag Patterns

Many subcommands share common flag patterns.

| Flag | Type | Purpose |
|------|------|---------|
| `--root` | string | Workspace root directory (persistent flag) |
| `--profile` | string | Filter repos by manifest profile |
| `--only` | []string | Target only specified repo IDs |
| `--skip` | []string | Exclude specified repo IDs |
| `--strategy` | string | Dirty tree handling (safe/stash/reset) |
| `--force` | bool | Allow destructive operations |
| `--json` | bool | Output in JSON format |
| `--dry-run` | bool | Show plan without executing |

## Validation Rules

The following validations are applied to the manifest (`workspace.yaml`):

- `version` must be `1`
- `name` is required
- Each repo's `id`, `url`, and `path` are required
- Repo IDs must be unique
- `path` and `repos_root` must be relative paths (absolute paths not allowed)
- Path traversal with `..` in `path` is not allowed

## CI/CD

The following are automatically run via GitHub Actions:

- **test**: `go build` → `go test -race -coverprofile` → coverage summary → artifact upload
- **lint**: Static analysis with `golangci-lint`

Releases use GoReleaser to generate binaries for Linux/macOS/Windows × amd64/arm64.

## Common Pitfalls

- When working with Git repos in tests, the bare repo's HEAD may point to an unintended branch. `CreateBareRepoWithBranch` includes logic to switch back to `main` before cloning.
- Writing directly to `os.Stdout` cannot be captured in tests. Always use `cmd.OutOrStdout()`.
- When running Git commands like `git init`, ensure the directory specified in `cmd.Dir` exists.
- `--strategy reset` includes a safety mechanism that refuses to execute without `--force`.

## Recommended GitHub Repository Settings

When setting up the GitHub repository for production use, we recommend:

- **Branch protection on `main`**:
  - Require pull request reviews before merging (at least 1 approval)
  - Require status checks to pass before merging (`test`, `lint`, `security`)
  - Require branches to be up to date before merging
  - Do not allow force pushes
- **Enable Dependabot** for automated dependency updates (configured in `.github/dependabot.yml`)
- **Enable GitHub Actions** for CI/CD (configured in `.github/workflows/`)
