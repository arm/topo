# Project Configuration

## Overview

Topo Projects support configuration through project parameters:

- `x-topo.parameters` defines parameter metadata (description, whether required, examples, and advisory hints)
- When a project parameter is used during an image build, its value is passed through standard Compose `build.args` and consumed by the Dockerfile as an `ARG`

## How Project Parameters Work

Projects extend [compose-spec](https://compose-spec.io/) with `x-topo.parameters` to define and document user-configurable project parameters.
Unless a service is intended for a remote processor, every service definition in these examples (and in compliant Projects) must include `platform: linux/arm64`. Remote processor services omit `platform` but must set `remoteproc` as their `runtime` so Implementations can recognize the exception.

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

## Worked example

### Creating a Project

Users can initialize a new Project, resulting in an empty `compose.yaml` file.

### Extending Projects with Project Services

When users extend a Project with services defined in another Project, Implementations must handle project parameter collection and validation. The examples below illustrate how a CLI interface might handle this:

**Interactive Mode**
Implementations may choose to prompt users when required parameters are missing:

```
$ topo extend ...

Missing Project Parameter
The greeting message to display in the container
GREETING (required)> Sup
```

**Direct Parameter Specification**
Implementations may support providing parameters directly:

```
$ topo extend ... GREETING=Sup`
```

**Non-Interactive Mode)**
Implementations may support a non-interactive mode that errors on missing required parameters:

```
$ topo extend ... --no-prompt

error: "Missing project parameters"
missing_parameters:
    GREETING:
      description: |
          The greeting message to display in the container
      required: true
      example: "Hello from Arm SME"
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

### Processing Steps

When users extend a Project with another Project, Implementations should perform the following steps:

1.  Retrieve the Project repository (e.g., clone to a local subdirectory)
1.  Parse the Project's `compose.yaml` and read parameter metadata from `x-topo.parameters`
1.  Collect values for any required parameters from `x-topo.parameters` (e.g., by prompting the user)
1.  Update the Project's `compose.yaml` with a service definition that extends the retrieved Project
1.  Set the `services.<service-name>.build.args` with the collected values when those parameters are consumed as Docker build arguments

### Resulting Configuration

After adding the service with project parameters, the compose file is updated:

**compose.yaml**

```yaml
services:
  cool-service:
    extends:
      file: ./cool-service/compose.yaml
      service: welcome
    build:
      args:
        GREETING: "Sup"
```

## Project Configuration Layering

### Projects Satisfying Requirements

Projects can provide default `build.args` values that satisfy requirements from Projects they extend. When a Project provides a value for a parameter that another Project marks as `required: true`, that parameter effectively becomes **optional** for end users.

**Extended Project's compose.yaml**

```yaml
x-topo:
  parameters:
    GREETING:
      description: The greeting message to display
      required: true
      example: "Hello, World"
```

**Parent Project's compose.yaml**

```yaml
services:
  welcome-service:
    extends:
      file: ./welcome-service/compose.yaml
      service: welcome
    build:
      args:
        GREETING: "Hello from Arm SME Project" # Satisfies the extended Project's requirement
```

This layering means:

- The extended Project declares `GREETING` as required
- The parent Project provides a default value
- End users can accept the default or override it
- The Project runs with plain `docker compose up` without prompting

### Projects Adding Configuration

Projects can also define their own `x-topo.parameters` to expose configuration:

```yaml
services:
  welcome-service:
    extends:
      file: ./welcome-service/compose.yaml
      service: welcome
    build:
      args:
        GREETING: "Hello, World"

  ollama-service:
    platform: linux/arm64
    build:
      context: .
      args:
        MODEL: qwen3:0.6B

x-topo:
  name: "My Project"
  parameters:
    MODEL:
      description: "hugging face model ID to use"
      required: true
      default: qwen3:7B
```

The effective required parameters for the Project are:

- Project-level parameters marked as `required: true`
- Extended Project parameters marked as `required: true` that the parent Project hasn't provided defaults for

When users create a new Project from an existing Project, Implementations should collect required parameters in the same manner as when extending an existing Project with another Project.
