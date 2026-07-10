# Speeding up image builds

Any image build with a slow, heavy step such as cloning large repos, installing hundreds of packages, compiling big dependency trees, downloading SDKs, fetching ML models etc., benefits from the same general set of caching strategies.

The goal is simple: avoid re-running expensive work that has not meaningfully changed.

## Layering

Container engines build in layers. Each layer is a diff in the image's filesystem on top of the previous layer.

As a rule of thumb, Dockerfile instructions such as `RUN`, `COPY`, and `ADD` create filesystem layers. Metadata-ish instructions such as `ENV`, `ARG`, `WORKDIR`, `CMD`, and `ENTRYPOINT` affect image config, but do not usually add filesystem contents themselves.

Layers are immutable, so if a layer changes, that layer and all subsequent layers are invalidated and must be rebuilt.

For example:

```text
layer[0] system packages
layer[1] dependencies
layer[2] source code build
layer[3] final packaging
```

If the source code changes, the source code build will be rerun, as will `layer[3]`. But `layer[0]` and `layer[1]` can still be reused.

As such, layers should be ordered from least often changed near the top of the file, to most often changed near the bottom:

```text
base image
system packages
SDK/toolchain setup
dependency manifests
dependency install
source code
build/test/package
```

System packages at the top, code near the bottom.

```dockerfile
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
RUN python -m build
```

Now `pip install` only reruns when `requirements.txt` changes, not every time some source file changes.

Avoid putting unrelated work in one layer. For example, a `RUN` layer that installs packages via `apt` and then runs a CMake build will lead to `apt` rerunning every time the source code changes invalidate that layer.

Prefer a small final image, but do not blindly assume the smallest base image gives the fastest build. Alpine base images are usually smaller, but may require more work in the image build stage to prepare the image for use or building.

Reputable sources often distribute performance and space-optimised build base images for different use cases, which are similar in size, or sometimes smaller than a DIY Alpine setup, and save a lot of setup compute.

Or, create and distribute your own base image. This allows CI to do the expensive work once, then lets users quickly use that layer for time-saving.

## Reduce the amount of work

The first thing to check for is expensive steps. Container engines report the time taken for build steps, so find the step eating the time.
Expensive steps increase build time and often layer size, which then increases time to transfer, pull, push, and start the image.

Aim to make expensive steps do less.

For example:

- Use `--no-install-recommends` with `apt`.
- Keep minimal pinned dependency lists.
- Use `--depth=1` for Git fetches.
- Use `-o=--depth=1` for `west` fetches.
- Use `--narrow` with `west` to avoid full refs/tags.
- Skip dependency groups you do not need.

For `apt`:

```dockerfile
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       build-essential \
       cmake \
       ninja-build
```

For Git:

```sh
git clone --depth=1 <repo>
```

For west/Zephyr:

```sh
west update --narrow -o=--depth=1
```

For west configs, you can ignore unneeded module groups:

```sh
west config manifest.group-filter -- "-hal,-Nordic,-ci" # replace with a filter for your use-case.
```

## BuildKit cache mounts

Builds are isolated from the host filesystem, and environments are temporary. This is generally beneficial for reproducibility; however, by default, it loses the benefits of tool-level caching.

Using `--mount=type=cache` allows the container engine to use a persistent build cache for tools in layers such as `apt`, `pip`, Go, Cargo, or even your build system's cache.

Unlike normal layer caching, this still helps when the `RUN` instruction has to execute again. The layer might be invalidated, but the cache directory is still there, so the tool can reuse previously downloaded or compiled data.

Common examples:

```dockerfile
# apt
RUN --mount=type=cache,target=/var/cache/apt \
    --mount=type=cache,target=/var/lib/apt/lists \
    apt-get update \
    && apt-get install -y --no-install-recommends build-essential cmake ninja-build

# pip
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt

# Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Go build cache
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build ./...

# Cargo
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    cargo build
```

CMake + Ninja example:

```dockerfile
RUN --mount=type=cache,target=/src/build \
    cmake -S . -B /src/build -G Ninja \
    && cmake --build /src/build
```

Zephyr example:

```dockerfile
RUN --mount=type=cache,target=/git-cache \
    west update --path-cache /git-cache --narrow -o=--depth=1
```

Even if `west update` reruns, for example because the Zephyr version changed, `west update --path-cache` can reuse cloned repos from the cache mount rather than cloning everything from scratch.

This enables much faster incremental rebuilds, much closer to speed-of-light for local development.

Engine support:

- Docker/nerdctl with BuildKit: supported.
- Podman/Buildah: supported in some configurations, but behaviour can differ.

## Registry-backed build cache

Build cache metadata and layers can be pushed to a container registry so they can be shared across machines and CI runs.

This is useful for any expensive layer, especially when building a layer takes longer than downloading it.

Docker/nerdctl Compose example:

```yaml
services:
  myapp:
    build:
      context: .
      cache_from:
        - type=registry,ref=registry.example.com/myapp:buildcache
      cache_to:
        - type=registry,ref=registry.example.com/myapp:buildcache,mode=max
```

This usually requires a CI job, or a manual build, to populate the cache when expensive inputs change.

Compared with a pre-baked base image:

- A pre-baked base image is explicit and simple: `FROM registry.example.com/base:v1`.
- A registry-backed cache is more automatic and can cache intermediate build steps.
- A base image is easier to reason about.
- A registry cache can be more flexible for complex builds and branch-based CI.

How Docker/BuildKit and Buildah caching differ:

|                     | Docker/BuildKit                              | Buildah                      |
| ------------------- | -------------------------------------------- | ---------------------------- |
| What's cached       | Operation graph + layer blobs                | Full intermediate images     |
| Cache precision     | Per-operation input checksums                | Per-intermediate-image match |
| Cache size          | Compact, deduplicated                        | Heavier                      |
| Cross-build sharing | Better, same operation can share cache       | More limited                 |
| Format              | BuildKit-specific cache metadata + OCI blobs | Standard OCI images          |

Podman/Buildah caveats:

- Buildah distributed caching uses intermediate images rather than BuildKit's operation graph model.
- Cache references and tag support have historically differed from Docker/BuildKit.
- The first build with an empty cache has to rebuild everything to populate the cache.
- Test this in your actual CI rather than assuming Docker behaviour maps 1:1.

Example Podman command:

```sh
podman build \
  --cache-from registry.example.com/myapp/cache \
  --cache-to registry.example.com/myapp/cache \
  -t myapp .
```

Podman Compose can also support `cache_from` and `cache_to` in the build section, depending on version.

## Practical checklist

When a build is slow, ask:

1. Which step is slow?
2. Can that step do less work?
3. Is that step before or after frequently changing source code?
4. Are dependency manifests copied separately from source code?
5. Is unrelated work accidentally bundled into one layer?
6. Would a cache mount help the tool reuse downloads or compiled objects?
7. Is the build running on a persistent machine or ephemeral CI?
8. Would a pre-baked base image be simpler than relying on cache behaviour?
9. Would a registry-backed cache help CI reuse expensive layers?

For local development, good layer ordering and cache mounts usually give the best return.

For CI, use good layer ordering first, then consider either a pre-baked base image or a registry-backed build cache.
