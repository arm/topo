package main

import (
	"github.com/arm/topo/internal/project"
	"github.com/spf13/cobra"
)

var serviceRemoveCmd = &cobra.Command{
	Use:   "remove <compose-filepath> <service-name>",
	Short: "Remove a service from the compose file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		serviceName := args[1]
		return project.RemoveService(composeFilePath, serviceName)
	},
}

func init() {
	serviceCmd.AddCommand(serviceRemoveCmd)
}
