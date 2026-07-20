# Glossary

This glossary defines the core terms used in Topo documentation and command output.

## Host

A general-purpose system used to configure, **deploy** to, or communicate with one or more **targets**. A **host** may also act as a **target** in cases where the same system runs the deployed application.

## Target

Where **deployments** run. A **target** may be a **heterogeneous device** containing one or more **processing domains**, such as a **primary processor**, **remote processors**, and **accelerators**. A **target** may be distinct from the **host** or may reside on the same system. Currently, this must be, at minimum, a computer running Linux on AArch64 (linux/arm64) on the **primary processor**.

## Processing Domain

Logical groupings of related components (usually on the **target**). **Processing domains** organize applications and services in containers by where they are on a device and what they can run.

## Heterogeneous Device

A device that contains multiple kinds of processing capabilities, such as a **primary processor**, **remote processors**, and **accelerators**, across one or more **processing domains**.

## Primary Processor

The main general-purpose **processing domain** running the Linux operating system. It manages system resources, runs user-space applications and containers, and coordinates **remote processors** via frameworks such as **Remoteproc Runtime**.

## Primary OS

The primary operating system of the **target**. For Topo, the target's **primary OS** is the Linux environment accessed by SSH from the **host** to run containers and coordinate with **remote processors** and other **processing domains**.

## Remote Processor

A peer execution environment on a **target** with an independent firmware lifecycle, typically managed from the **primary OS** through Linux remoteproc.

## Accelerator

A workload-specific component, such as a GPU or NPU, accessed through drivers, runtimes, or APIs. Unlike a **remote processor**, an **accelerator** is not typically managed as an independent firmware lifecycle through Linux remoteproc.

## Dependency

A required component (library, service, or another app) that must be available for an application to function. Often also described as a prerequisite, can relate to **target** or **host**.

## Topo Project

A containerised sample project that provides a starting point for development. It includes application code, build configuration, and an **x-topo** metadata block describing required hardware **features** and configurable parameters. **Projects** are configured when cloned. Topo allows the discovery and **deployment** of **Topo Projects** for validation.

## X-Topo

A metadata extension defined within a Compose file that describes a **Project**'s purpose, hardware requirements, **features**, and configurable parameters. Enables tooling (such as Topo) to provide validation, filtering, and interactive setup.

## Feature

A declared hardware capability required or utilised by a **Project** (e.g., NEON support, GPU compute capability, specific **accelerator** support). Used for matching **Topo Projects** to compatible **targets**.

## Deploy

The process of building container images on the **host**, transferring them to the **target**, and starting services, typically via Docker Compose. If a service is configured to run on a **remote processor**, a runtime such as **Remoteproc Runtime** may be used.

## Deployment

The outcome of a successful **deploy** operation. A **deployment** includes all active services, containers, and (where applicable) firmware executing across one or more **processing domains**, and reflects the system as it is currently provisioned and operating.

## Incremental Build/Deploy

A **build** approach where only changed layers are rebuilt, and a **deploy** approach where only rebuilt layers are transferred to the **target**, reducing iteration time.

## Remoteproc Runtime

An Open Container Initiative (OCI) compliant [container runtime that deploys firmware to **remote processors** via the Linux remoteproc framework](https://github.com/arm/remoteproc-runtime). Allows firmware to be packaged, distributed, and executed using standard container tools (e.g., Docker, containerd, Podman).

## Firmware Container

A container image that packages a firmware binary. When run with **Remoteproc Runtime**, the firmware is loaded onto a **target**'s **remote processor** and then started, instead of executed as a typical Linux process.
