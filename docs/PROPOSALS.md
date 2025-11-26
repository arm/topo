# Proposed API changes

This document describes commands in Topo which are expected to change, or be added.

## Workflow

1. Proposed changes to the Topo Command-Line Interface begin here with a Pull Request.
1. Once the PR is merged to `main`, the proposal is considered agreed.
1. When a follow-up PR delivers the implementation, delete the corresponding section from this file as part of that PR so the document only reflects outstanding proposals.

## Changing Commands

The following commands are expected to change:

### check-health -> health

#### Changes

- name: check-health -> health
- remove remoteproc checking behaviour (this will be rolled into another command)

#### Expected Usage Output

```
Check that your system is ready to use Topo

Usage:
  topo health [flags]

Flags:
  -h, --help            help for health
      --target string   The SSH destination.
```

#### Expected Behaviour Example

```sh
$> topo health --target 192.168.0.1
Host
----
SSH: ✅ (ssh)
Container Engine: ✅ (docker, podman)

Target
------
Connected: ✅
Container Engine: ✅ (docker)
```

## Commands to add

The following additional commands are planned:
