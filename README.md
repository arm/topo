[![Maintainability](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/maintainability.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli) [![Code Coverage](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/test_coverage.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli)

# Topo CLI

A CLI tool to edit a `compose.yaml` file.

## Installation

1. **Build**:

```sh
go build ./cmd/topo
```

## Usage

```sh
# List supported Service Templates
./topo list-service-templates

# Add a service based on a Service Template to the compose file
./topo add-service <compose-filepath> <service-name> <source>
# Examples:
#   Using a built-in template:
./topo add-service compose.yaml my-service template:hello-world
#   Using a git repository:
./topo add-service compose.yaml my-service git:https://github.com/user/repo.git
./topo add-service compose.yaml my-service git:git@github.com:user/repo.git --ref develop

# Remove a service from the compose file
./topo remove-service <compose-filepath> <service-name>

# Get the project at the specified path
./topo get-project <compose-filepath>

# Initialise a project in the current directory
./topo init [--target <ssh-target>]

# Show the config metadata
./topo get-config-metadata

# Generate a Makefile for the project
./topo generate-makefile <compose-filepath> [--target <ssh-target>]

# Get containers info from the target
./topo get-containers-info [--target <ssh-target>]

# Show information about the board
./topo check-health [--target <ssh-target>]
```
* `compose-filepath` is a path to the `compose.yaml` file
* `service-name` is the name of the new service to be added or removed.
* `source` is the service source with a scheme prefix:
  * `template:<template-id>` - Use a built-in Service Template (see `list-service-templates`)
  * `git:<git-url>` - Clone a git repository as a service template (supports `--ref` flag for branches/tags)
* `--target` is the SSH destination. It might be a config host alias (as defined in your ~/.ssh/config) or an SSH destination (`user@host`). If not specified it uses the `TOPO_TARGET` environment variable.

### How to deploy
```bash
cd <your project area>
make
```
