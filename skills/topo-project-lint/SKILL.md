---
name: topo-project-lint
description: Check Topo Project metadata correctness. Use when validating compose.yaml x-topo metadata, README alignment, deployment success messages, or project parameter wiring.
---

# Topo Project Lint

Use this skill when the user asks to lint, validate, check, review, or fix Topo Project metadata correctness.

Before acting, read `references/topo-project-context.md` for shared Topo Project vocabulary, authoritative references, and validation expectations.

## Workflow

Run these checks in order. If the user asked you to fix issues, make the smallest safe changes and re-run the relevant checks after editing.

1. Confirm the current directory is a Project root.
2. Validate `compose.yaml` against the Topo Project Specification schema.
3. Check that `x-topo.name` and `x-topo.description` match `README.md`.
4. Check that `x-topo.deployment_success_message` exists and is useful.
5. Check that every `x-topo.parameters` entry is consumed by the corresponding Docker build.

## Checks

### 1. Project Root

- Require a root-level `compose.yaml`.
- Require a root-level `x-topo` block in `compose.yaml`.
- Require `x-topo.name` to be present. This is the schema-minimum signal that the directory is a Topo Project.
- If the check fails, report that the current directory is not a Project and stop before schema validation unless the user asked you to repair it.

### 2. Schema

- Fetch the current published schema before validating.
- Check whether a supported validator is already installed, in this order: `check-jsonschema`, `ajv`, `jsonschema`.
- Prefer `check-jsonschema` because it validates YAML directly and is available from Homebrew as `brew install check-jsonschema`; on supported Ubuntu releases it may be available as `apt install python3-check-jsonschema`.
- Do not install validators on the user's behalf. If no supported validator is installed, stop and tell the user to install one before continuing.
- Treat schema errors as blocking issues. Fix schema errors before judging higher-level metadata intent.
- `check-jsonschema` caches schemas by default. If fetching a schema with a floating tag (e.g. `main`), ensure you specify `--no-cache` to get the current version.

### 3. README Alignment

- Read `README.md` from the Project root.
- Compare `x-topo.name` with the README title or clearly named project heading. The values do not need to be byte-identical, but they must identify the same Project without ambiguity.
- Compare `x-topo.description` with the README's first useful project summary. The metadata should accurately describe the same behavior, hardware focus, and user-facing purpose as the README.
- Flag generic, stale, marketing-only, or contradictory metadata.
- Prefer updating `x-topo` from the README when the README is specific and current. Prefer updating the README only when the Compose services show the README is stale.

### 4. Deployment Success Message

- Require `x-topo.deployment_success_message` to be present.
- The message should tell the user what succeeded and how to observe or use the deployed Project next.
- Flag placeholder messages, empty strings, and messages that mention URLs, ports, commands, or files not supported by the Compose services or README.

### 5. Project Parameter Consumption

- For every key in `x-topo.parameters`, find the service or services whose `build.args` provide that parameter as a Docker build argument.
- For each matching service, resolve its Docker build context and Dockerfile path. Use `Dockerfile` in the build context when `build.dockerfile` is omitted.
- Require the Dockerfile to declare the argument with `ARG <NAME>` or `ARG <NAME>=<default>` in a stage where it is needed.
- Check that the parameter is actually used after declaration, for example in `RUN`, `ENV`, `LABEL`, `COPY --from`, or another instruction. A declared but unused `ARG` does not count as consumed.
- Flag `x-topo.parameters` keys that are not present in any `services.<service>.build.args` unless the service extends another Project where the parameter is intentionally supplied by the parent Project.
- Flag `services.<service>.build.args` entries that look user-configurable but are missing from `x-topo.parameters` when the user asked for a comprehensive lint.
- Do not require runtime-only environment variables to appear in `x-topo.parameters`; this check is only for Docker build arguments.

## Reporting

Report findings in the same order as the checks. Include file paths, line numbers when available, the validation command used, and whether each check passed or failed. If changes were made, summarize the files edited and the re-validation result.
