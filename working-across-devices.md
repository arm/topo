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

### When Docker images are not well structured

Docker build performance is extremely sensitive to how well an image is structured for layer caching. Docker reuses unchanged layers, and `topo deploy` transfers images via a registry, so both rebuild time and transfer time benefit when changes are isolated to later layers.

In the best case, a 10 GB image can be rebuilt and transferred with only a tiny amount of work if the only change is a 1 KB file in the final layer. In the worst case, changing a similarly small file in an early layer can invalidate everything that follows, forcing most or all of the image to be rebuilt and transferred again.

Anyone authoring or extending a [Topo Template](https://github.com/arm/topo-template-format) needs to understand this, because it is easy to introduce large performance regressions in the developer iteration loop.
