---
name: topo-project-optimize-deployment
description: Optimize Topo Project deployment or development iteration performance by measuring the current bottleneck and applying the highest-leverage Docker build improvement.
---

# Topo Project - Optimize Deployment

Use this skill when the user asks to make `topo deploy`, Docker builds, initial deployment, or development iteration faster for a Topo Project.

Before acting, read `references/topo-project-context.md` for shared Topo Project vocabulary and validation expectations. Read `references/docker-build-performance.md` before proposing or editing Dockerfile or Compose build changes.

## Workflow

1. Identify the Project root. Prefer the repository directory containing the root-level `compose.yaml`.
1. Read `./project-benchmarking.md` in the Project root if it exists. Create it when recording the first benchmark entry.
1. Ask whether the user wants to prioritize initial deployment time or iteration time.
1. If the user prioritizes iteration time, inspect the project and identify the most likely frequent edit path, such as Python source, C/C++ source, model files, dependency manifests, or configuration. Ask the user to confirm the change type to optimize for before editing.
1. Check whether `topo` is installed. If it is, ask the user for an SSH Target and benchmark with `topo deploy --target <ssh-target>`. If `topo` is unavailable or the user cannot provide a Target, benchmark with `docker compose build`.
1. Inspect `compose.yaml`, Dockerfiles, `.dockerignore`, dependency manifests, and source layout to find cache invalidation boundaries and expensive build steps.
1. Before editing, identify and write down the current bottleneck hypothesis from benchmark output. If using `topo deploy`, treat it as the authoritative end-to-end benchmark. If the build phase is suspected but the Topo output does not clearly show the cause, run `docker compose build --progress=plain` as a diagnostic and extract three facts: the slowest non-cached build step, the first unexpected cache miss, and the build context transfer size. Choose the optimization that directly addresses one of those facts. Do not make generic Dockerfile cleanup changes unless they are tied to the recorded bottleneck.
1. Treat `x-topo.parameters` defaults and Compose `build.args` defaults as product behavior, not ordinary benchmark tuning knobs. Do not change a default value, such as a model, dataset, precision mode, feature flag, or runtime option, as the first optimization attempt just because it makes the benchmark faster.
1. Choose one high-leverage optimization. Prefer the smallest change likely to improve the selected scenario.
1. Take a baseline. To do this, ensure the docker build cache for the build images is clear first. Record the baseline in `./project-benchmarking.md`.
1. Apply the change, then re-run the same benchmark path used for the baseline. Update the same `./project-benchmarking.md` entry with the after-change result.
1. Report the baseline, change, result, and remaining likely bottlenecks. Encourage the user to start a new chat session for the next incremental optimization.

## Benchmarking

- Use the same command, Project root, build arguments, and Target for baseline and after-change measurements.
- Prefer `topo deploy --target <ssh-target>` because it includes Project build, transfer through the peer-to-peer registry, and deployment to the Target.
- Use `docker compose build` as the fallback when there is no Topo Target or `topo` is unavailable.
- Capture enough output to identify slow steps and cache misses, not just total wall-clock time.
- If optimizing iteration time, run or request a representative edit before the after-change benchmark when practical.
- Clear Docker caches as necessary to establish representative timings. Ask for user permission to do this _once_, remembering their decision if they consent.

## Benchmark Log

Record benchmark results in `./project-benchmarking.md` in the Project root so future runs can compare against prior work. Append a new entry for each optimization attempt and use this format:

```markdown
## YYYY-MM-DD - <initial deployment|iteration> - <short optimization name>

- Benchmark command: `<exact command>`
- Project root: `<path>`
- Target: `<ssh-target or none>`
- Representative edit: `<change type or n/a>`
- Bottleneck hypothesis: <one sentence>
- Diagnostic facts: <slowest non-cached step, first unexpected cache miss, context transfer size, or other relevant facts>
- Optimization: <exact change made>
- Baseline: <duration and key output facts>
- After change: <duration and key output facts>
- Result: <delta or inconclusive>
- Caveats: <missing Target testing, unsupported build engine behavior, noisy timing, or none>
```

If an attempt stops before an after-change benchmark, still record the baseline, hypothesis, and blocker so the next run has context.

## Optimization Guidance

- For initial deployment time, prioritize reducing the amount of work, using smaller dependency sets, avoiding unnecessary downloads, and pre-baking stable expensive setup into base images when appropriate.
- For iteration time, prioritize isolating expensive layers from frequently edited files, copying dependency manifests before source, and moving app code late in the Dockerfile.
- Use BuildKit cache mounts for package managers, compiler caches, model downloads, Git object stores, or similar local rebuild accelerators when the build engine supports them.
- Use registry-backed build cache only when a shared registry and cache population workflow exist or the user explicitly wants CI or multi-host cache reuse.
- Avoid adding non-standard Compose keys. Build settings must remain valid Compose build configuration.
- Avoid changing user-facing Project defaults to improve benchmark results. A default argument change is acceptable only when the user explicitly asks for a product trade-off, or when the current default is demonstrably wrong and the replacement preserves intended behavior. If proposing such a change, call it out as a behavior/product change rather than a Docker build optimization.
- Do not introduce a large pre-baked base image without explaining the registry storage, rebuild, and publishing trade-offs.

## Reporting

Report the priority selected, benchmark command, bottleneck hypothesis, diagnostic fact that supported it, edited files, exact optimization made, measured before and after results, and why the chosen change was the smallest change likely to improve the selected benchmark. Include any caveats such as missing Target testing or unsupported build engine behavior.
