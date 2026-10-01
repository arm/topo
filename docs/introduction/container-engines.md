---
sidebar_position: 3
title: Install a container engine for Topo
description: Set up Docker or Podman to build images on the host and run containers on the target.
---

# Install a container engine for Topo

Topo uses a container engine to build images on the [host](glossary.md#host) and run containers on the [target](glossary.md#target). The target must run Linux on AArch64 (`linux/arm64`). Choose one engine for both systems:

- [Docker](#docker) is the default.
- [Podman](#podman) requires `--engine podman` or `TOPO_ENGINE=podman`.

## Check container engines with Topo health

After you install Topo, run the health check:

```sh
topo health --target [user@]host
```

`topo health` checks the selected container engine on both the host and target. It also checks the Compose provider on the host. For Podman, add `--engine podman`. Follow any recommended actions and run the health check again after each change.

See the setup instructions for your selected engine below.

## Docker

Install the following components:

- On the host, install the Docker command-line interface (CLI), a running Docker-compatible engine, and Docker Compose 2.21.0 or later as a Docker CLI plugin.
- On the target, install Docker Engine and the Docker CLI. Docker Compose is not required on the target.

### Choose a host installation

Topo recommends Docker Desktop where it is supported. Otherwise, use the alternative for your host:

| Host          | Recommendation                                                                                                          |
| ------------- | ----------------------------------------------------------------------------------------------------------------------- |
| macOS         | [Docker Desktop](https://docs.docker.com/desktop/setup/install/mac-install/) or [Colima](#colima)                       |
| Linux x86_64  | [Docker Desktop](https://docs.docker.com/desktop/setup/install/linux/) or [Docker Engine](#docker-engine-on-linux)      |
| Linux Arm64   | [Docker Engine](#docker-engine-on-linux)                                                                                |
| Windows x64   | [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) or [Rancher Desktop](#rancher-desktop) |
| Windows Arm64 | [Docker Desktop](https://docs.docker.com/desktop/setup/install/windows-install/) (Early Access)                         |

The table shows Topo recommendations, not every platform that each container engine supports.

Review the [Docker Desktop license terms](https://docs.docker.com/subscription/desktop-license/) before installation. On Windows, use Linux containers.

#### Colima

Follow the [Colima installation instructions](https://colima.run/docs/installation/), including the steps to install the Docker CLI and [Docker Compose plugin](https://colima.run/docs/installation/#docker-compose-plugin). Use Colima's default Docker runtime.

#### Docker Engine on Linux

Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for your distribution.

Also install the [Docker Compose plugin](https://docs.docker.com/compose/install/linux/) and complete the [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/) so your user can run `docker` without `sudo`. Access to the Docker daemon grants [root-level privileges](https://docs.docker.com/engine/security/#docker-daemon-attack-surface).

#### Rancher Desktop

Follow the [Rancher Desktop installation instructions](https://docs.rancherdesktop.io/getting-started/installation/). Select **dockerd (moby)** as the container engine. Topo does not require Kubernetes.

### Install Docker on the target

The target must run Linux on AArch64 (`linux/arm64`). Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for the target distribution, then complete the [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/) so the target SSH user can run `docker` without `sudo`.

For a custom Linux distribution built with the Yocto Project, see [`meta-virtualization`](https://layers.openembedded.org/layerindex/branch/master/layer/meta-virtualization/).

## Podman

### Select Podman

Use `--engine podman` consistently for health checks and deployment commands:

```sh
topo health --engine podman --target user@target.example
topo deploy --engine podman --target user@target.example
topo ps --engine podman --target user@target.example
topo stop --engine podman --target user@target.example
```

Alternatively, set `TOPO_ENGINE=podman` in your environment. An explicit `--engine` flag overrides this setting. The selected engine applies to both the host and target.

Podman deployments do not support Compose services with a `runtime:` setting, including projects that require Remoteproc Runtime. Use Docker for those projects.

### Install Podman on the host

Install [Podman](https://podman.io/docs/installation) and Docker Compose as a `docker-compose` executable on `PATH`. Topo uses this provider through `podman compose`; `podman-compose` is not a substitute.

On macOS and Windows, initialise and start a [Podman machine](https://docs.podman.io/en/latest/markdown/podman-machine.1.html). On Linux, ensure the host Podman API socket is running using the [socket setup below](#enable-the-podman-api-socket).

### Install Podman on the target

Install [Podman](https://podman.io/docs/installation) so that the SSH user can run `podman` without `sudo`. Compose is not required on the target. Enable the API socket as described below.

#### Enable the Podman API socket

Topo queries the target Podman API from the host through a temporary SSH tunnel. Being able to run `podman` on the target is not enough: its API socket must also be functional and accessible to the SSH user.

For rootless Podman on a systemd-based target, log in **as the SSH user** and run:

```sh
systemctl --user start podman.socket
```

This enables socket activation of the Podman API service. To enable the socket on future user-session starts, use `systemctl --user enable podman.socket`. Use the same setup on a Linux host running rootless Podman.

If `topo health --engine podman --target user@target.example` reports:

```text
✗ Podman API (host-side Podman could not query the target API through topo’s temporary SSH tunnel)
```

Check the socket on the target as the SSH user:

```sh
systemctl --user status podman.socket
podman info --format '{{.Host.RemoteSocket.Path}}'
```

A typical rootless socket path is `/run/user/1000/podman/podman.sock`; the user ID can differ. Topo discovers the path rather than assuming UID 1000. Ensure the reported socket is functional and accessible to that user, then rerun the health check.

For systems without systemd, rootful Podman, or keeping a user service available after logout, follow the [Podman API service documentation](https://docs.podman.io/en/latest/markdown/podman-system-service.1.html). Do not make the socket world-accessible: API access grants full control of that user's containers. The target SSH server must also allow the forwarding used by Topo's tunnel.
