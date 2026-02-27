# agentws

[![CI](https://github.com/fbkclanna/agentws/actions/workflows/ci.yml/badge.svg)](https://github.com/fbkclanna/agentws/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![GitHub Release](https://img.shields.io/github/v/release/fbkclanna/agentws)](https://github.com/fbkclanna/agentws/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/fbkclanna/agentws)](https://goreportcard.com/report/github.com/fbkclanna/agentws)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/fbkclanna/agentws/pulls)

> [日本語](README.ja.md)

A CLI tool for reproducibly setting up local workspaces across multiple repositories that make up a product.

When combined with coding agents such as Codex or Claude Code, it unifies multiple repos under a single workspace and ensures agents launch from the correct directory.

## Features

- **One-command clone/sync** of the correct set of repos for each product
- Share the same workspace configuration across a team, preventing manual discrepancies and omissions
- Reproducible environments via `workspace.lock.yaml` that records exact commits (ideal for bug reproduction, verification, and code reviews)
- Selectively fetch only the repos you need using `profile` (e.g., exclude heavy analytics repos)

## Installation

### Quick Install (Linux / macOS)

```sh
curl -fsSL https://github.com/fbkclanna/agentws/releases/latest/download/agentws-install.sh | sh
```

To install a specific version or to a custom directory:

```sh
VERSION=0.2.0 INSTALL_DIR=~/.local/bin \
  curl -fsSL https://github.com/fbkclanna/agentws/releases/latest/download/agentws-install.sh | sh
```

### Go install

```sh
go install github.com/fbkclanna/agentws/cmd/agentws@latest
```

### GitHub Releases

Download platform-specific binaries from the [Releases page](https://github.com/fbkclanna/agentws/releases) (Linux / macOS / Windows, amd64 / arm64).

**Requirements:**

- Git 2.25 or later
- Go 1.26 or later (if using `go install`)

### Uninstall

If installed via `agentws-install.sh` or GitHub Releases, remove the binary:

```sh
sudo rm /usr/local/bin/agentws
```

If you specified a custom `INSTALL_DIR`, remove from that directory instead:

```sh
rm ~/.local/bin/agentws
```

If installed via `go install`:

```sh
rm $(go env GOPATH)/bin/agentws
```

## Quick Start

```sh
# 1) Create a workspace (interactively add repositories)
agentws init <workspace-name>

# 2) Sync repos
agentws sync

# 3) Check status
agentws status
```

## Directory Layout

By default, the workspace is created under the `--root` directory (e.g., `./products`).

```
products/
└── foo/
    ├── workspace.yaml
    ├── workspace.lock.yaml
    ├── AGENTS.md
    ├── CLAUDE.md -> AGENTS.md
    ├── docs/
    │   └── agentws-guide.md
    └── repos/
        ├── backend/
        ├── frontend/
        ├── infra/
        └── analytics/
```

## Commands

### `init <name>`

Creates a new workspace and generates `workspace.yaml`, `AGENTS.md`, and `CLAUDE.md` (a symlink to `AGENTS.md`).

When run without options, it launches interactive mode where you can enter repository URLs and branches one by one. It automatically infers repo IDs and paths from URLs and detects remote default branches. Press Enter with an empty URL to add a local repository (no remote).

```sh
agentws init foo
# Remote repository:
# ? Enter Git repository URL (empty for local): git@github.com:org/backend.git
#   → id: backend, path: repos/backend
# ? Branch: main
# ? Add another repository? Yes
#
# Local repository (press Enter with empty URL):
# ? Enter Git repository URL (empty for local): [Enter]
# ? Enter repository name (ID): config
#   → id: config, path: repos/config (local)
# ? Add another repository? No
```

To create from an existing manifest file, use `--from`:

```sh
agentws init foo --from git@github.com:org/workspaces.git#foo.yaml
```

**Options:**

| Option | Description |
|--------|-------------|
| `--root <dir>` | Root directory for the workspace (e.g., `./products`) |
| `--from <src>` | Import a manifest (e.g., local path or `repo#path` format) |
| `--force` | Overwrite even if a workspace already exists (use with caution) |

### `add [url ...]`

Adds repositories to an existing workspace. Supports both CLI and interactive modes.

```sh
# CLI mode: add one or more repos
agentws add https://github.com/org/backend.git
agentws add https://github.com/org/api.git --id api-service --ref develop --tag core

# Local repo mode: create a local repository (no remote URL)
agentws add --local my-service
agentws add --local my-service --path custom/dir --ref main --tag core
agentws add --local my-service --sync   # git init + initial commit

# Interactive mode: run without URLs
agentws add
```

When no URLs are provided and stdin is a TTY, interactive mode launches (same interface as `init`). You can also add local repositories by pressing Enter with an empty URL.

**Options:**

| Option | Description |
|--------|-------------|
| `--local` | Create a local repository (no remote URL). Args are treated as IDs |
| `--id <string>` | Repository ID override (single URL only) |
| `--path <string>` | Repository path override (single repo only) |
| `--ref <string>` | Git ref to checkout (default: auto-detected, or `main` for local) |
| `--tag <string>` | Tags to assign (repeatable) |
| `--sync` | Clone/initialize repositories immediately after adding |
| `--json` | Output added repositories as JSON |

### `sync`

Clones, fetches, and checks out repos according to `workspace.yaml` to bring the workspace in sync. Idempotent (designed to produce consistent state regardless of how many times it runs).

```sh
agentws sync
```

**Common options:**

| Option | Description |
|--------|-------------|
| `--profile <name>` | Select repos by profile |
| `--jobs <n>` | Number of parallel workers (e.g., `8`) |
| `--only <id1,id2>` | Sync only specified repos |
| `--skip <id1,id2>` | Exclude specified repos |

**Reproducibility (lock):**

| Option | Description |
|--------|-------------|
| `--lock` | Check out commits pinned in `workspace.lock.yaml` (reproducibility mode) |
| `--update-lock` | Update lock after sync |

**Handling dirty working trees:**

| `--strategy` | Description |
|--------------|-------------|
| `safe` (default) | Skip dirty repos |
| `stash` | Stash changes and continue |
| `reset` | Force reset (requires `--force`) |

**Safety/destructive operations:**

| Option | Description |
|--------|-------------|
| `--force` | Allow destructive operations (use with `reset`, etc.) |

### `status`

Displays the workspace status at a glance.

- Not cloned / cloned
- Current HEAD
- Dirty detection
- Differences from lock/manifest (if any)

```sh
agentws status
```

**Options:**

| Option | Description |
|--------|-------------|
| `--json` | JSON output (for CI integration) |

### `pin`

Pins the current HEAD of each repo to `workspace.lock.yaml` (records commits).

```sh
agentws pin foo
```

### `branches`

Lists the current branch, HEAD commit, and working tree state (dirty) for each repository in the workspace. Useful for quickly checking the state of each repo during cross-repo development.

```sh
agentws branches
```

**Example output:**

```
REPO        BRANCH                      HEAD        DIRTY
backend     feature/ABC-123-search-v2   a1b2c3d     false
frontend    feature/ABC-123-search-v2   c3d4e5f     true
infra       main                        9f8e7d6     false
analytics   (detached)                  deadbeef    false
```

**Options:**

| Option | Description |
|--------|-------------|
| `--json` | Output in JSON format |
| `--profile <name>` | Target only repos in the specified profile |
| `--only <id1,id2>` | Target only specified repos |
| `--skip <id1,id2>` | Exclude specified repos |

> **Notes:**
> - When `BRANCH` shows `(detached)`, the repo is checked out at a specific commit (e.g., from lock sync or commit-specified checkout).
> - `DIRTY=true` indicates uncommitted changes (including tracked and untracked files).

### `checkout --branch <branch>`

Switches all target repos in the workspace to the same branch name.

- If the branch exists locally → checks it out directly
- If it doesn't exist locally but exists on the remote → creates a tracking branch and checks it out
- If it doesn't exist anywhere → creates a new branch or skips depending on flags

```sh
agentws checkout --branch feature/ABC-123-search-v2
```

**Options:**

| Option | Description |
|--------|-------------|
| `--create` | Create the branch if it doesn't exist |
| `--from <ref>` | Starting point for new branches (overrides `base_ref`) |
| `--profile <name>` | Target only repos in the specified profile |
| `--only <id1,id2>` | Target only specified repos |
| `--skip <id1,id2>` | Exclude specified repos |
| `--strategy safe\|stash\|reset` | Dirty tree handling (default: `safe`) |
| `--force` | Allow destructive operations (use with `reset`) |
| `--dry-run` | Show target repos and planned actions without executing |

### `start <ticket> [slug]`

Generates a branch name following naming conventions from a ticket ID and creates & checks out the branch across all target repos in the workspace. Enables one-command setup for cross-repo feature development.

```sh
agentws start ABC-123 search-v2
# => Creates & checks out feature/ABC-123-search-v2
```

**Options:**

| Option | Description |
|--------|-------------|
| `--prefix feature\|bugfix\|hotfix` | Branch type (default: `feature`) |
| `--from <ref>` | Starting point for new branches (overrides `base_ref`) |
| `--profile <name>` | Target only repos in the specified profile |
| `--only <id1,id2>` | Target only specified repos |
| `--skip <id1,id2>` | Exclude specified repos |
| `--strategy safe\|stash\|reset` | Dirty tree handling |
| `--force` | Allow destructive operations |
| `--dry-run` | Show generated branch name and target repos without executing |

> **Notes:**
> - If a remote branch with the same name already exists, it will be checked out (creating a tracking branch).
> - For repos where the branch doesn't exist, a new branch is created from the `--from` reference or `origin/<base_ref>`.
> - Branch base resolution order: `--from` flag → `origin/<repo.base_ref>` → `origin/<defaults.base_ref>` → error.

### `doctor`

Runs diagnostics on the development environment. Reports errors if issues are found.

```sh
agentws doctor
```

**Checks performed:**

- Git installation verification
- Git version check
- SSH authentication check (`ssh -T git@github.com`)
- Connectivity test for each repo URL in the workspace (`git ls-remote`)

### `run -- <command>`

Runs a command in the workspace root directory. Everything after `--` is executed as-is.

```sh
agentws run -- make test
agentws run -- docker compose up -d
```

## Manifest: `workspace.yaml`

`workspace.yaml` declares the repos and rules that make up a product workspace.

### Example

```yaml
version: 1
name: foo
description: Foo product workspace
repos_root: repos

profiles:
  core:
    include_tags: [ "core" ]
  full:
    include_tags: [ "core", "infra", "data" ]

defaults:
  depth: 50
  partial_clone: false
  sparse_checkout: false
  base_ref: main

repos:
  - id: backend
    url: git@github.com:org/foo-backend.git
    path: repos/backend
    ref: main
    tags: [ "core" ]
    depth: 50
    post_sync:
      - name: "go mod download"
        workdir: "."
        cmd: [ "go", "mod", "download" ]

  - id: analytics
    url: git@github.com:org/foo-analytics.git
    path: repos/analytics
    ref: main
    tags: [ "data" ]
    base_ref: develop
    partial_clone: true
    sparse:
      - "pipelines/"
      - "docs/"

  - id: config
    local: true
    path: repos/config
    ref: main
    tags: [ "core" ]
```

### Fields

#### workspace

| Field | Description |
|-------|-------------|
| `version` (required) | Currently `1` |
| `name` (required) | Workspace name |
| `description` | Description |
| `repos_root` | Root directory for repos (default: `repos`) |

#### defaults

| Field | Description |
|-------|-------------|
| `depth` | Shallow clone depth (e.g., `50`) |
| `partial_clone` | Blobless clone (`--filter=blob:none` equivalent) |
| `sparse_checkout` | Default for sparse checkout |
| `base_ref` | Default branch base for `start`/`checkout --create` (branch name only, e.g., `main`) |

#### profiles

| Field | Description |
|-------|-------------|
| `include_tags` | Include repos with specified tags |
| `include_repo_ids` | Explicitly include repo IDs |
| `exclude_repo_ids` | Explicitly exclude repo IDs |

#### repo

| Field | Description |
|-------|-------------|
| `id` (required) | Logical name (must be unique) |
| `url` | Git URL (required for remote repos, must be empty for local repos) |
| `local` | `true` for local repositories (no remote URL) |
| `path` (required) | Clone destination (relative path; absolute paths and `..` are prohibited) |
| `ref` | Branch/tag/commit (defaults to `main` if omitted) |
| `base_ref` | Branch base for `start`/`checkout --create` (overrides `defaults.base_ref`) |
| `tags` | Tags for profile filtering |
| `required` | `true`/`false` (defaults to `true` if omitted) |
| `depth`, `partial_clone`, `sparse` | Per-repo settings |
| `post_sync` | Commands to run after sync (array). `cmd` is specified as an array (safe, no shell expansion) |

## Lock: `workspace.lock.yaml`

`workspace.lock.yaml` records the actual commits that were synced, ensuring reproducibility.

### Example

```yaml
version: 1
name: foo
generated_at: "2026-02-15T12:34:56+09:00"
tool_version: "0.1.0"

repos:
  backend:
    url: git@github.com:org/foo-backend.git
    ref: main
    commit: "a1b2c3d4..."
  analytics:
    url: git@github.com:org/foo-analytics.git
    ref: main
    commit: "deadbeef..."
```

**Usage:**

```sh
# Create/update the lock
agentws sync --update-lock

# Sync to commits pinned in the lock
agentws sync --lock
```

## Safety (Important)

To prevent writes outside the workspace, the CLI prohibits the following in `workspace.yaml`'s `repos_root` / `repos[].path`:

- Absolute paths
- Paths containing `..`

Additionally, destructive operations like `--strategy reset` should be used in conjunction with `--force`.

## Typical Workflows (Team Usage Examples)

### 1) Distribute from a workspace definition repo

Place `foo.yaml` in a repo dedicated to workspace definitions (e.g., `org/workspaces`). Team members sync with:

```sh
agentws init foo --from git@github.com:org/workspaces.git#foo.yaml
agentws sync --profile core
```

### 2) Pin commits for bug reproduction or verification

```sh
agentws sync --lock
```

### 3) Update lock for verification

```sh
agentws sync --update-lock
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

> 日本語のコントリビュートガイドは [CONTRIBUTING.ja.md](CONTRIBUTING.ja.md) をご覧ください。

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for details.

```sh
# Build
go build ./cmd/agentws

# Test
go test -race ./...

# Lint
golangci-lint run
```

## License

MIT License. See [LICENSE](LICENSE) for details.
