# Topo CLI

Compose, parameterize, and deploy containerized examples for Arm hardware.

Topo connects to a remote Arm board over SSH, discovers what it can do, and helps you make the most of it. It detects hardware capabilities, installs companion runtimes like [remoteproc-runtime](https://github.com/arm/remoteproc-runtime) to unlock features that aren't enabled out of the box, and deploys containerized workloads tailored to your specific board — all from a single CLI on your host machine.

## Core Concepts

### Host and Target

Topo operates across two machines:

- **Host machine** — your laptop, workstation, or CI runner where you run the `topo` CLI. It connects to the target over SSH and builds container images locally.
- **Target machine** — a remote Arm Linux board (e.g. Raspberry Pi, custom SoC, cloud Graviton instance) reachable over SSH. Topo deploys and runs containerized workloads on this machine.

Every command that touches the target accepts a `--target` flag with an SSH destination (`user@host` or an SSH config alias). Set `TOPO_TARGET` once in your environment to skip repeating it:

```sh
export TOPO_TARGET=pi@my-board
```

If you are working directly on an Arm Linux board, you can set `--target localhost` to use the same machine as both host and target.

### Target Description

Running `topo describe` SSHs into the target, probes the CPU model, core count, ISA features (NEON, SVE, SVE2, etc.), and any remoteproc heterogeneous processors, then writes the results to a `target-description.yaml` file. This file is used to match your board to compatible templates.

### Templates

A Topo template is a reusable Docker Compose service definition packaged with Arm-specific metadata — which CPU features it requires, what build arguments it accepts, and a short description. Templates can come from the built-in catalog (`template:Name`), a git repository (`git:https://...`), or a local directory (`dir:path`).

The template format is defined in [arm/topo-template-format](https://github.com/arm/topo-template-format).

## Installation

### Prerequisites

- Go (1.25)

Build from source:

```sh
go build ./cmd/topo
```

## Getting Started

### Check the status of the host and target systems

```sh
./topo health --target my-board
```

The `--target` flag accepts SSH config host aliases or `user@host` destinations. You can also set the `TOPO_TARGET` environment variable to avoid repeating this flag.

### Generate a description of your target hardware

```sh
./topo describe --target my-board
```

This creates a `target-description.yaml` in the current directory. This can be used to parameterize topo templates based on the hardware characteristics of your target.

### Create a new project

```sh
./topo init
```

This creates a `compose.yaml` in the current directory.

### Add a service to your project

List available templates:

```sh
./topo templates
```

Extend the compose file using a built-in template:

```sh
./topo extend compose.yaml template:Topo-Welcome
```

### Deploy to your target

```sh
./topo deploy --target my-board
```

## Usage

For detailed command information and all available options:

```sh
./topo --help
./topo <command> --help
```
