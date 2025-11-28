package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/spf13/cobra"
)

var examplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "List available Example Projects",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		return catalog.PrintExampleProjectRepos(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(examplesCmd)
}
