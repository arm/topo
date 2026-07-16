---
sidebar_position: -1
---

# What is a Topo Project?

A Topo Project is a containerized sample project for Arm-based Linux systems. At minimum, it is a directory containing a `compose.yaml`, Dockerfiles, and source code, with an `x-topo` metadata block that describes what the Project does, what hardware features it requires, and what parameters a user can configure.

This specification defines the `x-topo` extension. It was developed for use with [Topo CLI](https://github.com/arm/topo/), but this is an open spec and any tool can read and act on `x-topo` metadata to discover, validate, and deploy Projects.

Not sure what these terms mean? [Topo's glossary](../introduction/glossary.md) defines many of its core concepts.

## How It Works

A Project's `compose.yaml` is a standard [Compose](https://compose-spec.io/) file with an `x-topo` block at the root:

```yaml
services:
  app:
    platform: linux/arm64
    build:
      context: .
      args:
        GREETING: "Hello, World"

x-topo:
  name: "hello-world"
  description: "A simple greeting app for Arm"
  features: ["NEON"]
  parameters:
    GREETING:
      description: "Message shown by the app"
      required: true
      example: "Hello from Arm"
```

Because this is valid Compose, any Project can be run with plain `docker compose`. The `x-topo` block is what allows tools like Topo to add interactive configuration, parameter validation, and target feature matching on top.

## Specification

- Human-readable spec:
  - [Project Overview](00-overview.md)
  - [Authoring Topo Projects](01-authoring-projects.md)
  - [Project Configuration](02-project-configuration.md)
  - [Schema Compliance](03-schema.md)
- Machine-readable schema:
  - [`schema/topo-project-specification.json`](schema/topo-project-specification.json)

## Discover Topo Projects

A curated project catalog can be found either via:

- the [topo projects](https://github.com/arm/topo#2-find-a-topo-project) command or
- the [Topo Project Catalog](https://github.com/arm/topo-project-catalog/blob/main/data/catalog.json).

We welcome any contributors who wish to add their own Project to the project catalog to submit a Pull Request as indicated below.

## Propose Your Project to Topo

If you want your Project to be added to the project catalog:

1. Review the [Authoring Topo Projects section of the Specification](01-authoring-projects.md).
2. [Validate Schema Compliance](./README.md#validate-schema-compliance) of your proposed Project.
3. Open a Pull Request in the `Topo Project Catalog` repository to update the [catalog sources](https://github.com/arm/topo-project-catalog/blob/main/data/github_sources.json).

### Validate Schema Compliance

The [machine-readable schema](schema/topo-project-specification.json) to check any Project against is provided as a [JSON schema](https://json-schema.org/) and is therefore compatible with any supported tooling.

For validation workflows, see:

- [Validating Schema Compliance in Your Editor](03-schema.md#validating-schema-compliance-in-your-editor)
- [Validating Schema Compliance using CLI](03-schema.md#validating-schema-compliance-using-cli)

## Versioning

This format follows Compose-style evolution and does not require strict schema-version pinning by implementations.

Implementations should follow [Compose guidance for optional attributes](https://github.com/compose-spec/compose-spec/blob/main/01-status.md#requirements-and-optional-attributes).

| Metadata |                  |
| -------- | ---------------: |
| Status   | Work in progress |
| Created  |       2025-11-10 |
