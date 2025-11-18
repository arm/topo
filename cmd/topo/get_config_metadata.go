package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var getConfigMetadataCmd = &cobra.Command{
	Use:   "get-config-metadata",
	Short: "Show config metadata",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		return core.PrintConfigMetadata(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(getConfigMetadataCmd)
}
