# Topo

[![Go Report Card](https://goreportcard.com/badge/github.com/arm/topo)](https://goreportcard.com/report/github.com/arm/topo)

Discover and deploy containerised software to Arm hardware over SSH.

Topo matches your system with [Topo Templates](https://github.com/arm/topo-template-format) that showcase its capabilities. Already have a Docker Compose project? Topo gives you fast, incremental deployment to remote targets.

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
