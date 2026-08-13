# Topo Project Authoring Best Practices

Recommendations for delivering the best and most consistent user experience with Topo Projects

## Be semantically correct and leverage `x-topo` attributes as appropriate

`x-topo` contains attributes that help users discover and use your Project. Ensure you have considered all available attributes in the schema and used them as appropriate.

The [Topo project-authoring skills](https://github.com/arm/topo#project-authoring-skills) can help you lint and improve your Project.

## Only require `topo deploy` to build and run

Running `topo deploy` should be sufficient to build and start the application. Define every required build, dependency-fetching, and setup step in `compose.yaml` or its Dockerfiles, or provide the result in a referenced container image. Do not require users to run additional commands manually.

[Multi-stage builds](https://docs.docker.com/build/building/multi-stage/) can be used for compilation, dependency fetching, code generation, and asset bundling. Copy only the resulting artifacts and runtime dependencies into the final stage. This ensures `topo deploy` performs the complete build while keeping build tools, caches, and other build-only files out of the runtime image.

### Be fast to build and iterate

The best Topo Projects are fast to build, deploy, and iterate on. See [Build Optimization](04-build-optimization.md) or use the [`topo-project-optimize-deployment` skill](https://github.com/arm/topo#project-authoring-skills) for guidance on specific performance best practices.
