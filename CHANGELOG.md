# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-16

### Added

- **CLI with 10 subcommands**: `init`, `sync`, `status`, `pin`, `branches`, `checkout`, `start`, `doctor`, `run`
- **`workspace.yaml` manifest**: declarative workspace configuration with repos, profiles, defaults, and post-sync hooks
- **`workspace.lock.yaml`**: commit pinning for reproducible environments
- **Profile-based filtering**: select repos by tags or IDs using `--profile`, `--only`, `--skip`
- **Parallel sync**: concurrent repo operations with `--jobs` flag and progress display
- **Dirty tree strategies**: `safe` (skip), `stash`, `reset` handling for uncommitted changes
- **Interactive init**: guided repository setup with URL auto-detection and default branch discovery
- **`--from` import**: initialize workspace from local files or remote `repo#path` references
- **Cross-repo branching**: `checkout` and `start` commands for unified branch management across repos
- **Environment diagnostics**: `doctor` command checks Git, SSH, and repo connectivity
- **Install script**: `curl | sh` installer for Linux and macOS
- **GoReleaser**: automated binary builds for Linux/macOS/Windows on amd64/arm64
- **CI/CD**: GitHub Actions for testing, linting, and release automation

[0.1.0]: https://github.com/fbkclanna/agentws/releases/tag/v0.1.0
