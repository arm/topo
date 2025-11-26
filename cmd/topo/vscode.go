package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/vscode"
	"github.com/spf13/cobra"
)

var getProjectCmd = &cobra.Command{
	Use:    "get-project <compose-filepath>",
	Short:  "Print the project as JSON",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		return vscode.PrintProject(os.Stdout, composeFilePath)
	},
}

var getConfigMetadataCmd = &cobra.Command{
	Use:    "get-config-metadata",
	Short:  "Show config metadata",
	Hidden: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		return vscode.PrintConfigMetadata(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(getProjectCmd)
	rootCmd.AddCommand(getConfigMetadataCmd)
}
