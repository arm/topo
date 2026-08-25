---
sidebar_position: 2
---

# Install Docker for Topo

Topo uses Docker to build container images on the [host](glossary.md#host) and run containers on the [target](glossary.md#target). Install the following components:

- On the host, install the Docker command-line interface (CLI), a running Docker-compatible engine, and Docker Compose 2.21.0 or later as a Docker CLI plugin.
- On the target, install Docker Engine and the Docker CLI. Docker Compose is not required on the target.

## Check container engines with Topo health

After you install Topo, run the health check:

```sh
topo health --target [user@]host
```

`topo health` checks the container engine on both the host and target. It also checks Docker Compose on the host. Follow any recommended actions and run the health check again after each change.

- For a host container engine or Docker Compose error, [choose a host installation](#choose-a-host-installation).
- For a target container engine error, [install Docker on the target](#install-docker-on-the-target).

## Choose a host installation

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

### Colima

Follow the [Colima installation instructions](https://colima.run/docs/installation/), including the steps to install the Docker CLI and [Docker Compose plugin](https://colima.run/docs/installation/#docker-compose-plugin). Use Colima's default Docker runtime.

### Docker Engine on Linux

Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for your distribution.

Also install the [Docker Compose plugin](https://docs.docker.com/compose/install/linux/) and complete the [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/) so your user can run `docker` without `sudo`. Access to the Docker daemon grants [root-level privileges](https://docs.docker.com/engine/security/#docker-daemon-attack-surface).

### Rancher Desktop

Follow the [Rancher Desktop installation instructions](https://docs.rancherdesktop.io/getting-started/installation/). Select **dockerd (moby)** as the container engine. Topo does not require Kubernetes.

## Install Docker on the target

The target must run Linux on AArch64 (`linux/arm64`). Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for the target distribution, then complete the [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/) so the target SSH user can run `docker` without `sudo`.

For a custom Linux distribution built with the Yocto Project, see [`meta-virtualization`](https://layers.openembedded.org/layerindex/branch/master/layer/meta-virtualization/).
