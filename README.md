[![Maintainability](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/maintainability.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli) [![Code Coverage](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/test_coverage.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli)

# Topo CLI

A CLI tool to edit a `compose.topo.yaml` file.

## Installation

1. **Build**:

   ```bash
   go build ./cmd/topo
   ```

## Usage

```bash
# Show supported templates
./topo list-templates

# Add a service to the compose file
./topo add-service <compose-filepath> <template-id> [<service-name>]

# Remove a service from the compose file
./topo remove-service <compose-filepath> <service-name>

# Get the project at the specified path
./topo get-project <compose-filepath>

# Initialise a project at the specified path
./topo init-project <project-path> <project-name> [--target <ssh-target>]

# Show the config metadata
./topo get-config-metadata

# Generate a Makefile for the project
./topo generate-makefile <compose-filepath> [--target <ssh-target>]

# Get containers info from the board
./topo get-containers-info [--target <ssh-target>]
```

* `compose-filepath` is a path to the `compose.topo.yaml` file
* `project-filepath` is a path to the directory where a project will be created
* `template-id` is the id of the template to add.
* `service-name` is the name of the new service to be added (equal to `template-id` by default) or removed.
* `project-name` is the name of the project.
* `--target` is the SSH destination. It might be a config host alias (as defined in your ~/.ssh/config) or an SSH destination (`user@host`). If not specified it uses the `TOPO_TARGET` environment variable.

### How to deploy
```bash
cd <your project area>
make
```