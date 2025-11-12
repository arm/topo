package main

import (
	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var removeServiceCmd = &cobra.Command{
	Use:   "remove-service <compose-filepath> <service-name>",
	Short: "Remove a service from the compose file",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		composeFilePath := args[0]
		serviceName := args[1]
		return core.RemoveService(composeFilePath, serviceName)
	},
}

func init() {
	rootCmd.AddCommand(removeServiceCmd)
}
