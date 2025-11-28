[![Maintainability](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/maintainability.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli) [![Code Coverage](https://qlty.sh/badges/50b07af7-90e1-41a9-88c4-2533beb04d2b/test_coverage.svg)](https://qlty.sh/gh/Arm-Debug/projects/topo-cli)

# Topo CLI

Compose, parameterize, and deploy containerized examples for Arm hardware.

## Installation

Build from source:

```sh
go build ./cmd/topo
```

## Getting Started

### Create a new project

```sh
./topo init
```

This creates a `compose.project.yaml` in the current directory.

### Add a service to your project

List available templates:

```sh
./topo service templates
```

Add a service using a built-in template:

```sh
./topo service add compose.project.yaml my-service template:cortexa-welcome
```

### Deploy to your target

```sh
./topo deploy --target my-board
```

The `--target` flag accepts SSH config host aliases or `user@host` destinations. You can also set the `TOPO_TARGET` environment variable to avoid repeating this flag.

## Usage

For detailed command information and all available options:

```sh
./topo --help
./topo <command> --help
```
