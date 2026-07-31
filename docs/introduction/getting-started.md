# Getting started

After you [install Topo](install.mdx), use this guide to check a [target](glossary.md#target), discover a compatible [Topo Project](glossary.md#topo-project), and deploy the project.

:::note

The commands in this guide use `user@target.example` as an example SSH destination. Replace it with your target's SSH destination, such as an SSH alias, `user@hostname`, or `ssh://user@hostname:2222`.

:::

## Check the host and target

Run the health checks:

```sh
topo health --target user@target.example
```

Topo checks the required [host](glossary.md#host) tools, SSH connection, target container engine, and target hardware. Resolve all errors and review each warning. When available, the report includes commands to correct problems.

If the target's SSH host key is unknown, confirm the target identity. Then run the suggested command to accept the key. If public-key authentication is not configured, run the following command after you accept the host key:

```sh
topo setup-keys --target user@target.example
```

This command creates a dedicated key, copies the public key to the target, and updates your SSH configuration. The target must permit password authentication for the initial key transfer. If it does not, configure a public key through your existing device management process.

Run `topo health` again and confirm that no checks report an error. You can ignore a [Remoteproc Runtime](glossary.md#remoteproc-runtime) warning if you do not plan to deploy firmware to a [remote processor](glossary.md#remote-processor).

## Find a compatible Topo Project

List the available projects and check their compatibility with the target hardware:

```sh
topo projects --target user@target.example
```

The output includes a clone command for each project.

## Clone and configure a project

Clone the project:

```sh
topo clone https://github.com/Arm-Examples/topo-welcome.git
```

When you run the `topo clone` command, Topo prompts for values that configure the project. You can also pass values through the command line or a configuration file.

## Deploy the project

Change to the project directory and deploy it:

```sh
cd topo-welcome
topo deploy --target user@target.example
```

Topo builds or pulls the images on the host, transfers them to the target, and starts the services. Run commands that manage the deployment from the project directory. These commands use the `compose.yaml` file in the current directory.

## Verify the deployment

List the running containers for the project:

```sh
topo ps --target user@target.example
```

Check the `Address` column in the `topo ps` output for accessible service endpoints.

After you change the project, run `topo deploy --target user@target.example` again. Docker and Topo reuse cached layers where possible to reduce deployment time.

## Stop the deployment

From the project directory, stop the services:

```sh
topo stop --target user@target.example
```

This command stops the services but does not remove their containers.

## Next steps

- Run `topo <command> --help` for command options.
- Review the [Topo glossary](glossary.md) for core terms.
- Read the [Topo Project Specification](../project-specification/README.md) to create or adapt a project.
