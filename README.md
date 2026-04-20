# Topo

[![Go Report Card](https://goreportcard.com/badge/github.com/arm/topo)](https://goreportcard.com/report/github.com/arm/topo)

Discover and deploy containerised software to Arm hardware over SSH.

Point Topo at any Arm-based Linux device to discover software templates which showcase its capabilities. Pick one and Topo helps you configure it for your use case, then deploys it in minutes. The result? A standard Docker Compose project to learn from, modify, or use as a starting point for your own work.

Already have a Compose project? Topo gives you a fast, incremental build-deploy loop over SSH.

## Who is this for?

**You just got a board and want to see what it can do.** Topo scans your target and finds [Topo Templates](https://github.com/arm/topo-template-format) that showcase its capabilities, from running an LLM to comparing SIMD performance. Each one deploys in minutes and is a real Compose project you can learn from or build on.

**You want a faster edit-build-deploy loop.** Build on your laptop and deploy to a Pi or Jetson over SSH. Rebuilds are incremental, so after the first deploy you're often iterating in seconds.

**You have a heterogeneous device and want to use all of it.** Your board has coprocessors like a Cortex-M that normally need separate toolchains and manual firmware loading. Topo and [remoteproc-runtime](https://github.com/arm/remoteproc-runtime) let you orchestrate the whole device as one Docker Compose project.

## What does it look like?

```sh
# Check your target is ready
topo health --target pi@raspberrypi

# See which templates match your hardware
topo templates --target pi@raspberrypi

# Clone one and configure it for your target
topo clone https://github.com/Arm-Examples/topo-welcome.git
cd topo-welcome/
topo deploy --target pi@raspberrypi
```

## Highlights

- **Fast, incremental deploys** over SSH, with layer caching to keep rebuilds quick
- **Hardware-aware template discovery** that matches your target's actual capabilities
- **Real Compose projects** you configure at clone time, then inspect, modify, and reuse
- **Standard tooling throughout**: Docker Compose, container images, and OCI registries
- **Whole-device orchestration** of Linux services and coprocessor firmware in a single Compose project

## Installation

### Prerequisites

**Host machine** (where you run `topo`):

- [Docker](https://docs.docker.com/get-docker/)

**Target machine** (the remote Arm system):

- Reachable with SSH
- Linux on ARM64
- Docker

The host and target can be the same system. If you're working directly on an Arm Linux system, use `--target localhost`.

### Linux and macOS

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

### 2. Find a template

```sh
topo templates --target [user@]host
```

### 3. Clone your chosen template

Choose a template you wish to try, then clone it:

```sh
topo clone https://github.com/Arm-Examples/topo-welcome.git
```

If the template requires build arguments, Topo will prompt you for them.

### 4. Deploy to your target

```sh
cd topo-welcome/
topo deploy --target [user@]host
```

Topo builds the container images on your host, transfers them to the target over SSH, and starts the services.

### 5. Review the deployment

Your project is now running on your target. See the template README for details.

### 6. Stop the deployment

When you're done, stop the running services:

```sh
topo stop --target [user@]host
```

## Other Commands

Run `topo <command> --help` for full usage details.
