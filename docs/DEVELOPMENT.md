# Development Guide

This guide covers the development workflow, tools, and conventions for contributing to topo-cli.

## Linting

The project uses [golangci-lint](https://golangci-lint.run/) for Go code quality checks.

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2

# Run linter and formatter checks
golangci-lint run

# Run linter/formatter with auto-fix
golangci-lint run --fix
```
