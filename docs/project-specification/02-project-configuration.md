---
sidebar_position: 2
title: Project configuration
description: Define project parameters and reference them through environment variables.
---

# Project Configuration

## Overview

[Topo Projects](../introduction/glossary.md#topo-project) support configuration through project parameters:

- [`x-topo.parameters`](../introduction/glossary.md#x-topo) defines parameter metadata (description, whether required, examples, and advisory hints)
- Compose environment variable references connect parameter names to their uses in the project
- `topo configure` saves parameter values in `.env.topo` without rewriting the Compose file
- When a project parameter is used during an image build, its value is passed through standard Compose `build.args` and consumed by the Dockerfile as an `ARG`

## How Project Parameters Work

Projects extend [compose-spec](https://compose-spec.io/) with `x-topo.parameters` to define and document user-configurable project parameters.
Unless a service is intended for a [remote processor](../introduction/glossary.md#remote-processor), every service definition in these examples (and in compliant Projects) must include `platform: linux/arm64`. Remote processor services omit `platform` but must set [`remoteproc`](../introduction/glossary.md#remoteproc-runtime) as their `runtime` so Implementations can recognize the exception.

**compose.yaml**

```yaml
services:
  welcome:
    platform: linux/arm64
    build:
      context: .
      # The fallback allows running without a configured value.
      args:
        GREETING: "${GREETING:-Hello, World}"

x-topo:
  name: "Topo Welcome"

  # Project parameter metadata for interactive prompting
  parameters:
    # Implementations prompt users to provide these values
    GREETING:
      description: |
        The greeting message to display in the container
      required: true
      example: "Hello from Arm SME"
```

These project parameters are then passed to the service's Dockerfile as standard Docker `ARG` values when the service uses Compose `build.args`.

### Example Dockerfile

**Dockerfile**

```Dockerfile
FROM nginx:alpine

ARG GREETING

# Docker files cannot require an arg - it is necessary to force failure if the value is not specified
RUN test -n "$GREETING" || (echo "ERROR: GREETING project parameter is required" && exit 1)
...
```

## Configure parameter values

From the project directory, run:

```sh
topo configure GREETING="Hello from Arm SME"
```

Topo saves the value in `.env.topo`. The Compose file continues to reference `${GREETING:-Hello, World}`.
To use the saved values with Docker Compose directly, specify the environment file:

```sh
docker compose --env-file .env.topo up --build
```

Parameter names must match their environment variable references. If the project defines parameters but none are referenced, `topo configure` rejects the project as using the legacy format.

## Migrate a legacy configured project for use with Topo 14.0.0

Topo 14.0.0 replaces configuration by rewriting literal build arguments with configuration through environment variable references.
Before configuring a legacy project, run:

```sh
topo configure --migrate-to-env
```

The migration moves current parameter values to `.env.topo` and replaces their uses in the Compose file with environment variable references.
Review the Compose file changes after migration. Then use `topo configure` to update parameter values.

## Parameter Hints

Parameter definitions may include `hints`, which Implementations can use to discover, filter, or suggest suitable parameter values. Hints do not define validation constraints, and Implementations may ignore hints they do not understand.

Hint keys must use lowercase dotted namespaces to avoid collisions, such as `model.task` and `meta.type`. Hint values may be strings, numbers, booleans, or arrays of those scalar values.

```yaml
x-topo:
  parameters:
    MODEL:
      description: "Model artifact reference"
      example: "bartowski/Qwen_Qwen3.5-0.8B-GGUF:SmolLM2-135M-Instruct-Q4_K_M.gguf"
      hints:
        meta.type:
          - huggingface.repo-id
        model.task: text-generation
        model.format: gguf
```

Recommended hint key conventions include:

- `meta.type` — suggests a type or set of types to identify valid values for this parameter, such as `huggingface.repo-id` or `git.uri`
- `model.task` — suggests a Hugging Face task or pipeline filter, such as `text-generation`
- `model.format` — suggests a desired artifact or file format, such as `gguf`
