# Topo

[![Go Report Card](https://goreportcard.com/badge/github.com/arm/topo)](https://goreportcard.com/report/github.com/arm/topo)

Discover what your Arm hardware can do and deploy workloads that use it to its full potential.

Every Arm system has a different mix of architecture features, and many include additional specialized processors alongside the main CPU. Topo connects to your target over SSH, probes its capabilities, and matches it to ready-made containerized workloads that fully leverage your specific hardware. Where a system has multiple processors, Topo can install companion runtimes like [remoteproc-runtime](https://github.com/arm/remoteproc-runtime) to extend container workloads across all of them.

## Core Concepts

### Host and Target

Topo operates across two machines:

- **Host machine** — your laptop, workstation, or CI runner where you run the `topo` CLI. It connects to the target over SSH and builds container images locally.
- **Target machine** — a remote Arm Linux system (e.g. Raspberry Pi, custom SoC, cloud Graviton instance) reachable over SSH. Topo deploys and runs containerized workloads on this machine.

Commands that connect to the target accept a `--target` flag with an SSH destination (`user@host` or an SSH config alias). Set `TOPO_TARGET` once in your environment to skip repeating it:

```sh
export TOPO_TARGET=user@my-board
```

If host and target are the same system, use `--target localhost`.

### Target Description

The `topo describe` command probes your board and writes a `target-description.yaml` that captures CPU features, core topology, and any heterogeneous processors.

### Templates

Topo templates extend the [Compose Specification](https://compose-spec.io/) popularised by Docker, adding `x-topo` metadata that declares CPU feature requirements and build arguments. Topo uses your target description to match and configure compatible templates for your board. Templates can come from the built-in catalog (`template:Name`), a git repository (`git:https://...`), or a local directory (`dir:path`).

The full format specification is at [arm/topo-template-format](https://github.com/arm/topo-template-format).

## Installation

For **Linux** and **macOS** users, the quickest way to install is with the installer script, which auto-detects your platform and places the binary on your `PATH`:

In your shell, run:
```sh
curl -fsSL https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.sh | sh
```

Or, To install a specific version or choose a custom install directory:
```sh
curl -fsSL https://raw.githubusercontent.com/arm/topo/refs/heads/main/scripts/install.sh | sh -s -- --version v4.0.0 --path ~/.local/bin
```

Alternatively, You may also want to manually install it from [GitHub Releases](https://github.com/arm/topo/releases/latest).

**Windows**

Download the latest Windows binary from [GitHub Releases](https://github.com/arm/topo/releases/latest).

1. Extract the `.zip` file you downloaded.
2. Open **Command Prompt** in the folder where `topo.exe` was extracted:
   - Right-click in the folder → **Open in Terminal**
3. Run the following commands:

```bat
mkdir "%USERPROFILE%\tools\topo" && move topo.exe "%USERPROFILE%\tools\topo"
setx PATH "%PATH%;%USERPROFILE%\tools\topo"
```
Then from any terminal run `topo --help to confirm installation`

### Prerequisites

**Host machine** (where you run `topo`):

- SSH client (`ssh`)
- [Docker](https://docs.docker.com/get-docker/)

**Target machine** (the remote Arm system):

- Linux on ARM64
- Docker
- `lscpu` (typically pre-installed; used for hardware probing)
- SSH server

The host and target can be the same system. If you're working directly on an Arm Linux system, use `--target localhost`.

## Getting Started

This walkthrough takes you from first connection to a running deployment. The examples use `my-board` as the SSH destination — replace it with your own `user@host` or SSH config alias, or set `TOPO_TARGET` once to skip repeating it:

```sh
export TOPO_TARGET=user@my-board
```

### 1. Check that everything is ready

```sh
topo health --target my-board
```

```
Host
----
SSH: ✅ (ssh)
Container Engine: ✅ (docker)

Target
------
Connectivity: ✅
Container Engine: ✅ (docker)
Hardware Info: ✅ (lscpu)
Remoteproc Runtime: ⚠️ (remoteproc-runtime not found on path)
  → run `topo install remoteproc-runtime`
Remoteproc Shim: ⚠️ (containerd-shim-remoteproc-v1 not found on path)
  → run `topo install remoteproc-runtime`
Subsystem Driver (remoteproc): ✅ (m4_0)
```

- ❌ must be resolved before continuing.
- ⚠️ can be resolved to unlock full functionality.
- ℹ️ are informational and won't block the core workflow.

### 2. Describe your target hardware

```sh
topo describe --target my-board
```

This SSHs into the target, probes CPU features, and writes a `target-description.yaml` in the current directory. Topo uses this file to match your system to compatible templates.

### 3. Find a template

```sh
topo templates --target-description target-description.yaml
```

This lists available templates and indicates compatibility with your target hardware.
If you don't already have a target description file for your board, you can still use:

```sh
topo templates --target my-board
```

### 4. Clone a template into a new project

```sh
topo clone template:topo-welcome
```

If the template requires build arguments, Topo will prompt you for them. You can also supply them on the command line:

```sh
topo clone template:topo-welcome GREETING_NAME="World"
```

This creates a project directory containing a `compose.yaml`, and any source files from the template.

### 5. Deploy to your target

```sh
cd my-project/
topo deploy --target my-board
```

Topo builds the container images on your host, transfers them to the target over SSH, and starts the services.

### 6. Stop the deployment

When you're done, stop the running services:

```sh
topo stop --target my-board
```

## Other Commands

The Getting Started walkthrough above covers the core flow. These additional commands are available:

| Command                      | When to use it                                                                                              |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `init`                       | Scaffold a new empty project instead of cloning a template                                                  |
| `extend`                     | Add services from a template into an existing project                                                       |
| `service remove`             | Remove a service from your compose file                                                                     |
| `setup-keys`                 | Set up SSH key authentication if your target currently uses password-based SSH, which Topo does not support |
| `install remoteproc-runtime` | Install the remoteproc runtime on your target                                                               |

Run `topo <command> --help` for full usage details.
