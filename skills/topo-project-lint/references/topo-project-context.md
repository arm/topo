# Shared Topo Project Context

Topo (https://github.com/arm/topo) is a CLI app which discovers, configures, builds, and deploys containerized examples to Arm-based Linux targets over SSH. A Topo Project (https://github.com/arm/topo/docs/project-specification) is a Compose project that remains runnable with plain `docker compose`, plus root-level `x-topo` metadata that lets Topo validate the Project, match it to target hardware features, and prompt for build-time configuration.

Use Topo vocabulary precisely:

- Host: the machine running Topo and building images.
- Target: the Arm64 Linux system where the deployment runs.
- Project: a containerized sample project containing `compose.yaml`, supporting files such as Dockerfiles and source, and `x-topo` metadata.
- X-Topo: the Compose extension key, written `x-topo`, that describes Project identity, type, hardware features, deploy message, and configurable project parameters.
- Feature: a Target hardware capability required or used by the Project, such as `NEON`, `SVE`, `SME`, a GPU, or an NPU.
- Remote processor: a peer execution environment managed from Linux, usually through Remoteproc Runtime and `runtime: remoteproc` services.
- Project parameters: Project configuration values, with prompts and validation described in `x-topo.parameters` and actual Docker build values carried through Compose `build.args` when needed.

Refresh spec-sensitive context at runtime before making or validating Project changes. Authoritative references, in order:

- Published Topo Project Specification schema: `https://raw.githubusercontent.com/arm/topo/refs/heads/main/docs/project-specification/schema/topo-project-specification.json`.
- Published Topo Project Specification docs: `https://github.com/arm/topo/docs/project-specification`, especially `README.md`, `00-overview.md`, `01-authoring-projects.md`, `02-project-configuration.md`, and `03-schema.md`.
- Published Topo glossary for domain terms: `https://github.com/arm/topo/blob/main/docs/introduction/glossary.md`.
- Compose Spec for standard Compose semantics. Do not invent non-standard Compose keys except the root-level `x-topo` extension.

When references conflict, prefer the schema for validation behavior, then the specification docs for authoring intent, then the target repository's actual Compose behavior for the smallest safe change.

When validating changes, check if `topo` is installed, and prompt the user for an SSH target to test against. `topo clone dir:./path/to/project` can be used to test project parameters, `topo deploy` can be used to test build and deploy.
