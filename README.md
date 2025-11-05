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
./topo add-service <compose-filepath> <service-template-id> [<service-name>]

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
* `service-template-id` is the id of the Service Template to add.
* `service-name` is the name of the new service to be added (equal to `service-template-id` by default) or removed.
* `--target` is the SSH destination. It might be a config host alias (as defined in your ~/.ssh/config) or an SSH destination (`user@host`). If not specified it uses the `TOPO_TARGET` environment variable.

### How to deploy
```bash
cd <your project area>
make
```
