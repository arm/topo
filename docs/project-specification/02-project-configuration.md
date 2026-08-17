# Project Configuration

## Overview

[Topo Projects](../introduction/glossary.md#topo-project) support configuration through project parameters:

- [`x-topo.parameters`](../introduction/glossary.md#x-topo) defines parameter metadata (description, whether required, examples, and advisory hints)
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
      # Optional default: allows running with plain docker compose
      # Not used by Implementations that read x-topo.parameters
      args:
        GREETING: "Hello, World"

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

## Parameter Hints

Parameter definitions may include `hints`, which Implementations can use to discover, filter, or suggest suitable parameter values. Hints do not define validation constraints, and Implementations may ignore hints they do not understand.

Hint keys must use lowercase dotted namespaces to avoid collisions, such as `huggingface.task` or `file.format`. Hint values may be strings, numbers, booleans, or arrays of those scalar values.

```yaml
x-topo:
  parameters:
    MODEL:
      description: "Model artifact reference"
      default: "bartowski/Qwen_Qwen3.5-0.8B-GGUF:SmolLM2-135M-Instruct-Q4_K_M.gguf"
      hints:
        huggingface.task: text-generation
        file.format: gguf
```

Recommended hint key conventions include:

- `huggingface.task` — suggests a Hugging Face task or pipeline filter, such as `text-generation`
- `file.format` — suggests a desired artifact or file format, such as `gguf`
