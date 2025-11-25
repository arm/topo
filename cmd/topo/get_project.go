package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/spf13/cobra"
)

var getProjectCmd = &cobra.Command{
	Use:   "get-project <compose-filepath>",
	Short: "Print the project as JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		return project.Print(os.Stdout, composeFilePath)
	},
}

func init() {
	rootCmd.AddCommand(getProjectCmd)
}
