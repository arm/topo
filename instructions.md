# topo ps Instructions

This document explains how to use `topo ps`, what it currently supports, and how to try it out.

## What `topo ps` Does

`topo ps` creates and maintains a local filesystem view of a target system topology under a directory.
By default, the directory name is the target passed with `--target` (for example `dxg/`).

It groups containers by subsystem:

- `Host`
- remote processors detected on target (for example: `m4`, `r5f0`, etc.)

The view auto-refreshes while `topo ps` is running.

Only **running** containers are included in the topology tree.

## Command

```bash
topo ps --target <user@host-or-ssh-alias> [--path <dir>] [--refresh-interval <duration>]
```

Flags:

- `--target`, `-t`: SSH target (or set `TOPO_TARGET`)
- `--path`: local output directory for the topology view (default: target name)
- `--refresh-interval`: poll interval (default: `2s`)

Stop `topo ps` with `Ctrl+C`.
`Ctrl+C` stops updates and command processing; the directory snapshot remains on disk.

## Filesystem Layout

`topo ps` creates this layout:

```text
<target-name>/
  README
  Host/
    <container-name>--<container-id>/
      id
      name
      state
      status
      image
      runtime
      command
      last_result
  <remoteproc-name>/
    <container-name>--<container-id>/
      ...
```

Notes:

- Container directory names use `<name>--<short-id>` to keep names readable and unique.
- `Host` is always top-level.
- Remote processor containers are under `<target-name>/<processor>/...`.
- `command` is write-driven control (see below).
- `last_result` contains `ok` or an error message from the last command execution.

## Supported Operations (Current)

Only these two container operations are supported:

- `start`
- `stop`

Write the operation into the container `command` file:

```bash
echo stop > <target-name>/Host/my-service--abc123def456/command
echo start > <target-name>/Host/my-service--abc123def456/command

# Remote processor example
echo stop > <target-name>/<processor>/my-rproc-service--abc123def456/command
```

Unsupported values are ignored and reported in `last_result`.

## Auto-Update Behavior

While `topo ps` is running, it periodically:

1. probes subsystem topology on target
2. reads container list + runtime metadata
3. updates local filesystem tree
4. removes stale container directories
5. processes requested `command` actions

This means after a new deploy, the topology tree updates automatically.

## Quick Try-Out

### 1. Start `topo ps`

```bash
topo ps --target <your-target>
```

Or if you already exported:

```bash
export TOPO_TARGET=<your-target>
topo ps
```

### 2. Inspect topology tree

```bash
find <target-name> -maxdepth 3 -type f | sort
```

### 3. Read container state

```bash
cat <target-name>/Host/<container-name>--<id>/state
cat <target-name>/Host/<container-name>--<id>/status
```

### 4. Stop/start a container

```bash
echo stop > <target-name>/Host/<container-name>--<id>/command
cat <target-name>/Host/<container-name>--<id>/last_result

echo start > <target-name>/Host/<container-name>--<id>/command
cat <target-name>/Host/<container-name>--<id>/last_result
```

### 5. Deploy something new and verify update

Run `topo deploy --target <your-target>` in your project, then check that new container directories appear under `<target-name>/`.

## Troubleshooting

- If the output directory stays empty, verify target connectivity:
  - `topo health --target <your-target>`
- If operations fail, inspect:
  - `last_result` file in the container directory
  - `topo ps` terminal logs
- If target is localhost:
  - use `--target localhost`
