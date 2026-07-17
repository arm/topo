# Development guide

This guide covers the development workflow, tools, and conventions for contributing to `topo`.

## Topo CLI

### Prerequisites

- Go 1.26+

### Building

```sh
go build ./cmd/topo
```

### Linting

The project uses [golangci-lint](https://golangci-lint.run/) for Go code quality checks.

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1

# Run linter and formatter checks
golangci-lint run

# Run linter/formatter with auto-fix
golangci-lint run --fix
```

### Testing

The project uses [Go's built-in support for unit testing](https://pkg.go.dev/cmd/go/internal/test) to provide test coverage.

```bash
# Run all tests
go test ./...
```

Some tests have a dependency on docker. Test container images are built automatically when needed.

> Note that if docker is missing, the dependent tests will just be skipped as opposed to failing.

#### Golden Files

A subset of the end-to-end tests use golden files to compare CLI output with known output. These tests fail whenever the output changes. An output change is not necessarily breaking. Review the [breaking change policy](breaking-changes.md) before accepting the change.

Update the golden files after confirming that the new output is compatible or that the change is set as breaking:

```
UPDATE_GOLDEN=1 go test ./e2e/...
```

## Skills

Public agent skills live under `skills/`.

To test skills from this checkout while developing them, install the local repository with `npx skills`. Choose symlinks when prompted if you want edits in this checkout to be reflected immediately.

```sh
npx skills add . --global
```

Each installable skill folder should be self-contained, but shared Topo Project context is maintained in `skills/_shared/topo-project-context.md` to avoid hand-edited drift across skills. After editing that shared context, update each skill's `references/topo-project-context.md` copy:

```bash
node scripts/sync-skill-context.js
```

Check that skill context references are current:

```bash
node scripts/sync-skill-context.js --check
```

Skill-specific instructions should stay workflow-focused. Put stable common vocabulary in the shared context, reference the current schema/docs for evolving spec details, and avoid copying the full specification into individual skills.
