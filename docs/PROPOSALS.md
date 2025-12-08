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

### service add -> extend

#### Changes

- name: `service add` -> `extend`
- support extending multi-service compose files
- use `--prefix` flag to customize service prefix and clone directory

#### Expected Usage Output

```
Extend a compose file with services from a template

Usage:
  topo extend <compose-file> <template> [flags] [-- ARG=VALUE ...]

Flags:
      --prefix string   Name for the cloned directory and service prefix (default: slugify(x-topo->name))
  -h, --help            help for extend
```

#### Expected Behaviour Example

```sh
$> topo extend compose.yaml git:Arm-Debug/llm-stack --prefix llm
# Clones template to ./llm/
# Adds to compose.yaml:

services:
  llm-service-1:
    extends:
      file: ./llm/compose.yaml
      service: service-1
  llm-service-N:
    extends:
      file: ./llm/compose.yaml
      service: service-N
```

- Without `--prefix`, the prefix defaults to a slugified version of the template's `name` field.
- ⚠️ A clashing target folder name or extended service name cancels the operation and reports error to the user.

### service remove -> 🗑️

#### Changes

- remove `service remove` command
