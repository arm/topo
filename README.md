# Topo

Discover and deploy containerised software to Arm hardware over SSH.

Point Topo at any Arm-based Linux device to discover Topo Projects that showcase its capabilities. Pick one and Topo helps you configure it for your use case, then deploys it in minutes. The result? A standard Docker Compose project to learn from, modify, or use as a starting point for your own work.

Already have a Compose project? Topo gives you a fast, incremental build-deploy loop over SSH.

## Who is this for?

**You just got a board and want to see what it can do.** Topo scans your target and finds [Topo Projects](https://github.com/arm/topo/docs/project-specification) that showcase its capabilities, from running an LLM to comparing SIMD performance. Each one deploys in minutes and is a real Compose project you can learn from or build on.

**You want a faster edit-build-deploy loop.** Build on your laptop and deploy to a Pi or Jetson over SSH. Rebuilds are incremental, so after the first deploy you're often iterating in seconds.

**You have a heterogeneous device and want to use all of it.** Your board has remote processors like a Cortex-M that normally need separate toolchains and manual firmware loading. Topo and [Remoteproc Runtime](https://github.com/arm/remoteproc-runtime) let you orchestrate the whole device as one Docker Compose project.

Not sure what these terms mean? The [glossary](docs/introduction/glossary.md) defines Topo's core concepts.

## What does it look like?

```sh
# Check your target is ready
topo health --target pi@raspberrypi

# See which projects match your hardware
topo projects --target pi@raspberrypi

# Clone one and configure it for your target
topo clone https://github.com/Arm-Examples/topo-welcome.git
cd topo-welcome/
topo deploy --target pi@raspberrypi
```

## Highlights

- **Fast, incremental deploys** over SSH, with layer caching to keep rebuilds quick
- **Hardware-aware project discovery** that matches your target's actual capabilities
- **Standard tooling throughout**: Docker Compose, container images, and OCI registries
- **Whole-device orchestration** of Linux services and remote processor firmware in a single Compose project

## Installation

### Prerequisites

**Host machine** (where you run `topo`):

- [Docker](https://docs.docker.com/get-docker/)
- curl
- OpenSSH Client

**Target machine** (the remote Arm system):

- Reachable with SSH
- Linux on ARM64
- Docker

The host and target can be the same system. If you're working directly on an Arm Linux system, use `--target localhost`.

### Linux and macOS

Using [Homebrew](https://github.com/arm/homebrew-topo):

```sh
brew install arm/topo/topo
```

Or use the install script:

```sh
curl -fsSL https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.sh | sh
```

### Windows

```sh
irm https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.ps1 | iex
```

Alternatively, manually add the appropriate binary from [GitHub Releases](https://github.com/arm/topo/releases/latest) to your `PATH`.

## Getting Started

### 1. Check that everything is ready

```sh
topo health --target [user@]host
```

### 2. Find a Topo Project

```sh
topo projects --target [user@]host
```

### 3. Clone your chosen Topo Project

Choose a Topo Project you wish to try, then clone it:

```sh
topo clone https://github.com/Arm-Examples/topo-welcome.git
```

If the project requires parameters, Topo will prompt you for them.

### 4. Deploy to your target

```sh
cd topo-welcome/
topo deploy --target [user@]host
```

Topo builds the container images on your host, transfers them to the target over SSH, and starts the services.

### 5. Review the deployment

Your project is now running on your target. See the project README for details.

### 6. Stop the deployment

When you're done, stop the running services:

```sh
topo stop --target [user@]host
```

## Other Commands

Run `topo <command> --help` for full usage details.

## Project Authoring Skills

This repository includes public agent skills that help authors create and validate Topo Projects.

- `topo-project-context`: provides Topo and Topo Project reference context for questions about `x-topo` metadata, schema, docs, and CLI Project behavior.
- `topo-project-bootstrap`: converts a repository into a Topo Project by adding or improving `compose.yaml` and `x-topo` metadata.
- `topo-project-lint`: reviews an existing Topo Project for correctness, consistency, and authoring best practices.
- `topo-project-optimize-deployment`: optimizes `topo deploy` or Docker build performance for initial deployment and iteration workflows.

### Installing Skills

You can install the skills with [`npx skills`](https://github.com/vercel-labs/skills):

```sh
npx skills add arm/topo
```

Or install the skills manually by copying or symlinking the directories under `skills/` into your agent's skills directory.

Restart your agent after installing or updating skills.
