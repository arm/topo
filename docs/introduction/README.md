# Overview

Topo is a command-line tool for discovering, configuring, and deploying containerized software to Arm-based Linux devices over SSH. It uses Compose projects, container images, and standard container tools.

You run Topo on a host and deploy to a target. The host can be a Linux, macOS, or Windows computer. The target is an Arm64 Linux device that runs the deployed services. An Arm64 Linux system can act as both the host and target.

This diagram shows where Topo runs, what it deploys, and how you iterate on a project.

![Topo host-to-target deployment and development loop](./img/topo-overview.svg)

## Why Topo?

Use Topo to:

- Evaluate an Arm device by finding projects that match its hardware capabilities
- Build on your host and use an incremental deployment loop with a remote device
- Orchestrate Linux services and, on supported heterogeneous devices, remote processor firmware in one project

Topo works with [Remoteproc Runtime](https://github.com/arm/remoteproc-runtime) to package and run firmware through container workflows.

## Projects

A Topo Project is a Compose project with `x-topo` metadata. The metadata describes the project purpose, hardware requirements, and configurable parameters. Topo uses this information to check target compatibility and configure the project for your use case.

The result remains a standard Compose project that you can inspect, modify, and use with existing container tools. You can also use Topo to deploy an existing Compose project whose Linux services target `linux/arm64`.

## Typical workflow

A typical workflow has 4 steps:

1. Run `topo health` to check the host, SSH connection, target software, and target hardware.
2. Run `topo projects` to list projects and check their compatibility with the target.
3. Run `topo clone` to copy and configure a project on the host.
4. Run `topo deploy` to build or pull images, transfer them to the target, and start the services.

Later deployments reuse cached image layers where possible. This reduces the work required after you change a project.

## Get started

Follow the [Getting Started](getting-started.mdx) guide to deploy a web application to a target. Alternatively, to create or publish a Topo Project from scratch, see the [Topo Project Specification](../project-specification).
