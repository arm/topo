# Development Guide

This guide covers the development workflow, tools, and conventions for contributing to `topo`.

## Linting

The project uses [golangci-lint](https://golangci-lint.run/) for Go code quality checks.

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1

# Run linter and formatter checks
golangci-lint run

# Run linter/formatter with auto-fix
golangci-lint run --fix
```

## Testing

The project uses [Go's built-in support for unit testing](https://pkg.go.dev/cmd/go/internal/test) to provide test coverage.

```bash
# Run all tests
go test ./...
```

Some tests have a dependency on docker and a smaller subset of these depend on the existence of a test container to act as topo target. This image can be built like so:

```bash
docker build -t topo-e2e-target:latest ./internal/testutil/test-container
```

> Note that if either of these test dependencies are missing, the dependent tests will just be skipped as opposed to failing.

### Golden Files

A subset of our e2e tests rely on "golden files" to assert CLI output against a known good state. These tests will fail when breaking changes are made and can be updated in place with the `UPDATE_GOLDEN` environment variable when running the tests.

```
UPDATE_GOLDEN=1 go test ./e2e/...
```
