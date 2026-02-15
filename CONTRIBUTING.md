# Contributing to agentws

> 日本語版は [CONTRIBUTING.ja.md](CONTRIBUTING.ja.md) をご覧ください。

Thank you for your interest in contributing to agentws! We welcome contributions of all kinds — bug reports, feature requests, documentation improvements, and code changes.

## How to Contribute

### Reporting Bugs

- Search [existing issues](https://github.com/fbkclanna/agentws/issues) to avoid duplicates.
- Use the [Bug Report template](https://github.com/fbkclanna/agentws/issues/new?template=bug_report.md) and include steps to reproduce, expected behavior, and environment details.

### Suggesting Features

- Open an issue using the [Feature Request template](https://github.com/fbkclanna/agentws/issues/new?template=feature_request.md).
- Describe the problem you're trying to solve and your proposed solution.

### Submitting Pull Requests

1. Fork the repository and create your branch from `main`.
2. Follow the development setup in [DEVELOPMENT.md](DEVELOPMENT.md).
3. Make your changes, ensuring tests pass:
   ```sh
   go build ./cmd/agentws
   go test -race ./...
   golangci-lint run
   ```
4. Write or update tests as needed.
5. Open a pull request with a clear description of the changes.

## Development Setup

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed instructions on building, testing, and linting.

## Coding Guidelines

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions focused and well-tested.
- Use `cmd.OutOrStdout()` for CLI output (never `os.Stdout` directly).
- Ensure path validation for any user-supplied paths (no absolute paths or `..` traversal).

## Commit Message Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix   | Usage                          |
|----------|--------------------------------|
| `feat:`  | New feature                    |
| `fix:`   | Bug fix                        |
| `docs:`  | Documentation only             |
| `test:`  | Adding or updating tests       |
| `chore:` | Maintenance, CI, dependencies  |
| `ci:`    | CI/CD configuration changes    |

Example: `feat: add --dry-run flag to checkout command`

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
