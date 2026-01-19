# Update Templates Script

This repository includes a Go script used to keep the templates list JSON up to date.

## Overview

The script regenerates the templates list JSON from a fixed set of repositories. It should be run whenever repositories are added, removed, or changed. Template definitions are sourced from each repository’s x-Topo metadata.

## Usage

Run the script from the root of the repository:

    go run ./scripts/update_templates

When executed, this command will:
- Iterate over the configured repository list
- Collect template metadata
- Regenerate and update the templates list JSON file

## Adding a Repository

To include a new repository in the templates list, update the `repoList` variable in the main:

    var repoList = []string{
        "topo-cortexa-welcome#main",
        "topo-kleidi-service#main",
        "STM32-Heteogenous-Communications-example#main",
        "topo-armv9-cpu-llm-chat#master",
        "topo-simd-visual-benchmark#master",
    }

### Format

Each entry must follow this format:

    <repository-name>#<branch>

Example:

    my-new-template-repo#main

After modifying the list, re-run the script to apply the changes.

Note that this only supports the arm-debug org right now

