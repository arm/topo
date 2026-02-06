package core

import (
	"fmt"
)

// PrintHelp prints CLI Help (lives in core for reuse by orchestration layer).
func PrintHelp() {
	binaryName := "topo"
	fmt.Print(binaryName + ` help

Usage:
  ` + binaryName + ` list-templates
  ` + binaryName + ` add-service <compose-filepath> <template-id> [service-name]
  ` + binaryName + ` remove-service <compose-filepath> <service-name>
  ` + binaryName + ` get-project <compose-filepath>
  ` + binaryName + ` init-project <project-path> <project-name> [<ssh-target>]
  ` + binaryName + ` get-config-metadata
  ` + binaryName + ` generate-makefile <compose-filepath> [<ssh-target>]
  ` + binaryName + ` get-containers-info [<ssh-target>]
  ` + binaryName + ` version

Commands:
  list-templates           List available service templates
  add-service              Add a service to the compose file
  remove-service           Remove a service from the compose file
  get-project              Print the project as JSON
  init-project             Initialise a new project
  get-config-metadata      Show config metadata
  generate-makefile        Generate a Makefile for the project
  get-containers-info      Show container info running on the board
  version                  Print version info
`)
}
