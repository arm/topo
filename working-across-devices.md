# Improving Topo usability across use cases and target/host mix

## Current architecture

Topo currently uses a model where:

- Image builds happen on the host
- Images are transferred to the target using a peer-to-peer container registry, leveraging layer caching
- The target is always `linux/arm64`
- Images must be `linux/arm64` if they are "standard" Linux containers, but can be any arch if using [remoteproc-runtime](https://github.com/arm/remoteproc-runtime)

## When the current architecture works poorly

### When the Target is more powerful than the host

If you're working on a relatively standard laptop but deploying to an Nvidia DGX Spark, Topo forces you to build your images on the slower device to transfer to the more powerful device.

**Slowdown factor:** Equal to the build performance difference between the Target and Host

### When the Host is not Arm64 and you're building a standard (i.e. non-remoteproc-runtime) container

Because the image must be built for the target architecture (i.e. `platform: linux/arm64`), the build must run under CPU architecture emulation.

**Slowdown factor:** Typically 2x-10x versus a native arm64 build, depending on how much CPU-bound work runs under emulation

### When the build depends upon the specifics of the architecture of the target

Because we build on the host, a container build cannot reliably observe the eventual target's CPU features at build time. This means we can't, for example, detect SVE acceleration inside the build and then confidently turn it on in a compile step.

In principle, some target-specific build configuration could instead be driven explicitly from `topo describe`, which probes the real target over SSH, rather than inferred implicitly inside the build.

That said, once a build starts selecting CPU-specific optimizations, the resulting image becomes more closely tied to the target class. This can reduce portability and cache reuse, and may require separate builds for different devices even when they are all `linux/arm64`.

### When good Docker image authoring requires too much expertise

Topo gets its fast edit-build-deploy loop from Docker and OCI primitives, but the quality of that loop depends heavily on how well images are authored. Techniques such as sensible layer ordering, multi-stage builds, and modern build cache features can make rebuilds and transfers dramatically faster, while poor image structure can cause very large regressions.

In the best case, a 10 GB image can be rebuilt and transferred with only a tiny amount of work if the only change is a 1 KB file in the final layer. In the worst case, changing a similarly small file in an early layer can invalidate everything that follows, forcing most or all of the image to be rebuilt and transferred again.

This creates an awkward requirement for Topo users: even when they are primarily trying to build for Arm hardware, they may need to understand advanced container build techniques to get good iteration performance.

## Possible solutions

### Option 1: Build on target via the container engine's SSH transport

Both Docker and Podman support talking to a remote daemon/service over SSH. The container engine tars up the local build context and sends it over the SSH connection. The build runs natively on the target, and the resulting image lands on the target. This means there is no need for rsync, no need for the peer-to-peer registry transfer, and no CPU emulation.

This approach solves three of the four problems at once:

- **Performance mismatch:** The build runs on the more powerful target natively.
- **Architecture mismatch:** No emulation needed; the build runs on real arm64 hardware.
- **Target-specific CPU features:** The build can observe the real CPU (SVE, etc.) since it runs on the target itself. Though the portability tradeoff described above still applies.

The image ending up on the target also eliminates the registry transfer step entirely, which is a significant simplification of the deploy pipeline for this path.

**What this looks like in Topo:**

A PoC has been implemented (Docker only) which demonstrates this working:

- `topo deploy --build-on target`
- Topo directs the container engine to talk to the target over the existing SSH connection
- Build context transfer, build execution, and image placement all happen through the engine's native machinery
- The peer-to-peer registry step is skipped since the image is already on the target
- The deploy pipeline reduces to three steps: build, pull, up (all on target)

**Requirements:** A container engine capable of building images must be available on the target. This is likely already the case for most Topo targets since they need a container runtime to run the deployed workloads.

#### Container engine compatibility

Docker and Podman both support remote builds over SSH, but the mechanisms differ:

- **Docker** uses `-H ssh://user@host` to direct the CLI at a remote Docker daemon. The local CLI handles context transfer transparently.
- **Podman** uses `podman --remote --connection <name>` or the `CONTAINER_HOST` environment variable, which includes a socket path (e.g. `ssh://user@host/run/user/1000/podman/podman.sock`). The socket path varies depending on whether Podman runs rootful or rootless.

The build toolchains also differ: Docker uses BuildKit, while Podman uses Buildah. Most Dockerfile features work in both, but caching behaviour and some advanced features (e.g. `RUN --mount=type=cache`) may behave differently.

Compose support varies too. `docker compose` is mature and tightly integrated. `podman compose` wraps either `docker-compose` or Podman's own implementation, with varying levels of build feature support.

Topo would need to detect which engine is available on the target and adjust its remote connection flags and build invocations accordingly.

#### Limitation: build context transfer

The container engine's SSH transport sends the *entire* build context (as a tar) to the remote daemon on every build. For small projects this is fine, but for projects with large assets in the build directory (e.g. multi-GB model weights), this becomes a bottleneck: the full context is re-sent every time, even if nothing changed.

This is a meaningful tradeoff compared to the current "build on host" path, where the build context never leaves the local machine and only changed layers are transferred via the registry.

Mitigations:

- **`.dockerignore`/`.containerignore`** becomes critical. Exclude anything the build doesn't actually need.
- **Keep large assets out of the build context.** If model weights or datasets are already on the target (or available from a URL, registry, or shared filesystem), use `ADD <url>`, volume mounts, or a multi-stage build that fetches them remotely. This keeps the context small regardless of approach.
- **rsync + remote build (Option 2)** would handle this better for large contexts, since rsync uses delta compression and only sends changed bytes.

### Option 2: rsync project to target, build remotely

rsync the project directory to the target, then run the container build on the target pointing at the synced directory.

- **Upside:** rsync uses delta compression, so only changed bytes are transferred. For projects with large assets that change infrequently (model weights, datasets), this is dramatically more efficient than re-sending the full build context each time.
- **Downside:** More moving parts than Option 1. Requires managing a remote working directory, handling cleanup, and ensuring Dockerfile `COPY` directives still resolve correctly against the synced path. Needs `.dockerignore`-equivalent filtering (or an explicit exclude list).

This approach could complement Option 1 rather than replace it: use the engine's SSH transport for small-context projects, and rsync + remote build for projects with large assets. A possible combined flow:

1. rsync the build context to the target (smart, incremental)
2. Build on the target pointing at the synced directory (native, fast)
3. Compose up on the target (image is already there)

#### Container engine compatibility

This option is more engine-agnostic than Option 1, since the build runs locally on the target rather than through a remote connection. As long as the target has a container engine that can build from a local directory (`docker build`, `podman build`), this works. The main compatibility concern is around compose support for the "up" step.

### Option 3: Automatic strategy selection

Rather than requiring a flag, Topo could choose the build location automatically based on what it already knows:

- Host is arm64, target is arm64, host is powerful → build locally (current behaviour)
- Host is x86, target is arm64 → build on target (avoid emulation)
- Target has significantly more cores/RAM than host → build on target

`topo describe` already probes the target over SSH, so the data is available. A `--build-on host|target|auto` flag could let users override the heuristic. This could be layered on top of any of the above options.

### Separately: reducing the Docker expertise requirement

The image authoring problem is somewhat orthogonal to where the build runs. Even with builds on the target, poor layer ordering still causes unnecessary rebuilds. Some ideas:

- **`topo lint` or `topo check`** that flags common Dockerfile/Containerfile antipatterns (e.g. copying large source trees before dependency install, missing `.dockerignore`/`.containerignore`)
- **Well-structured template Dockerfiles** that serve as examples of best practice
- **Warnings** when no ignore file exists, or when the build context is suspiciously large
