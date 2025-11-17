package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/spf13/cobra"
)

var listProjectsCmd = &cobra.Command{
	Use:   "list-projects",
	Short: "List available Projects",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return catalog.PrintExampleProjectRepos(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(listProjectsCmd)
}
