# Feature: topo ps command

## Background

- `topo describe --target <target>` creates a target description file that contains information about the heterogenous processors (and their features) on the system.
- An example of a target description file produce through `topo decribe` is this [one](./target-description.yaml).
- After deploying, `docker compose ps` has the information of the running containers for the system.
- There is already a vscode extension feature in place that can show the topology of the system (processors) and the containers running on each of them.

## Feature: topo ps command

Introduce a new command `topo ps --target <target>` that creates and exposes a Virtual Filesystem (VFS) under `topology` that shows the different processors in the target and the containers running on each of them.

The VFS should support operations like stopping a running container, etc. Deduce the operations from the codebase, but only implement 2 operations.

For example, to stop a running container in <processor1> called <process 1>, the user should run something like:

```
echo "stop" > topology/<processor1>/<process 1>
```

The latter is an overall example and should not be taken by heart, implement according to VFS best practices and what is relevant for the topo project.
The VFS should be automatically updated: If a new deployment is performed, the filesystem that was only showing the processors before should now show the containers running in those processors (after the deployment operation), of course.
