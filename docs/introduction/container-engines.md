---
sidebar_position: 2
---

# Install Docker for Topo

Topo uses Docker to build container images on the [host](glossary.md#host) and run containers on the [target](glossary.md#target). Install the following components:

- On the host, install the Docker command-line interface (CLI), a running Docker-compatible engine, and Docker Compose 2.21.0 or later as a Docker CLI plugin.
- On the target, install Docker Engine and the Docker CLI. Docker Compose is not required on the target.

Configure the host user and target SSH user to run `docker` commands without `sudo`.

:::caution

On Linux, membership in the `docker` group grants root-level privileges. Review the [Docker daemon security guidance](https://docs.docker.com/engine/security/#docker-daemon-attack-surface) before you grant access.

:::

## Install Docker on the host

Choose one of the following installation methods. Docker Desktop is the simplest option. Rancher Desktop, Colima, and Docker Engine provide open source alternatives for supported systems.

### Docker Desktop

[Docker Desktop](https://docs.docker.com/desktop/) includes Docker Engine, the Docker CLI, and Docker Compose. It is available for supported macOS, Windows, and Linux systems.

:::note

Review the [Docker Desktop license terms](https://docs.docker.com/subscription/desktop-license/) before installation. Some organizations and types of commercial use require a paid subscription.

:::

Follow the [Docker Desktop installation instructions](https://docs.docker.com/desktop/setup/install/) for your host. On Windows, configure Docker Desktop to use Linux containers.

Start Docker Desktop before you use Topo, and keep it running. Signing in to Docker Desktop is optional unless your organization requires it. Sign in to [increase Docker Hub pull limits, access private images, or apply organization security policies](https://docs.docker.com/desktop/setup/sign-in/).

### Rancher Desktop

[Rancher Desktop](https://rancherdesktop.io/) is an open source desktop application for macOS, Windows, and Linux. It includes the Docker CLI and Docker Compose.

:::note

On Windows, install [Windows Subsystem for Linux 2 (WSL 2)](https://aka.ms/wslinstall) before you install Rancher Desktop.

:::

Follow the [Rancher Desktop installation instructions](https://docs.rancherdesktop.io/getting-started/installation/) for your host. When Rancher Desktop starts for the first time, select **dockerd (moby)** as the container engine. Topo does not require Kubernetes. You can change these settings later in **Preferences**.

On macOS and Linux, select **Automatic** to add the Rancher Desktop tools to `PATH`. If you select **Manual**, add `~/.rd/bin` to `PATH` yourself.

Start Rancher Desktop before you use Topo, and keep it running.

### Colima

[Colima](https://colima.run/) provides container runtimes in a Linux virtual machine on macOS and Linux. Colima uses the Docker runtime by default, but you must install the Docker CLI and Docker Compose separately.

Follow the [Colima installation instructions](https://colima.run/docs/installation/). To install Colima and the required Docker tools with Homebrew, run:

```sh
brew install colima docker docker-compose
mkdir -p ~/.docker/cli-plugins
ln -sfn "$(brew --prefix)/opt/docker-compose/bin/docker-compose" ~/.docker/cli-plugins/docker-compose
```

Start Colima with the Docker runtime:

```sh
colima start
```

Keep Colima running while you use Topo. Do not start Colima with the `containerd` runtime.

### Docker Engine on Linux

[Docker Engine](https://docs.docker.com/engine/) provides a native, open source container engine for Linux without a desktop application.

Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for your Linux distribution. Install the [Docker Compose plugin](https://docs.docker.com/compose/install/linux/) if your installation method does not include it.

Complete the relevant [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/). Configure Docker to start when the host starts, and enable your user to run `docker` commands without `sudo`.

## Install Docker on the target

The target must run Linux on AArch64 (`linux/arm64`). Follow the [Docker Engine installation instructions](https://docs.docker.com/engine/install/) for the target distribution.

Complete the relevant [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/). Configure Docker to start when the target starts, and enable the SSH user to run `docker` commands without `sudo`.

For a custom Linux distribution built with the Yocto Project, use the [`meta-virtualization`](https://layers.openembedded.org/layerindex/branch/master/layer/meta-virtualization/) branch that matches your Yocto Project release.

## Verify installation

Run the Topo health checks for the host and target:

```sh
topo health --target user@target.example
```

Replace `user@target.example` with the SSH destination for your target.

A successful Docker installation reports:

- `Container Engine: ✅` under both `Host` and `Target`
- `Docker Compose: ✅` under `Host`
