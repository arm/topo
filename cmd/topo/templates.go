package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/spf13/cobra"
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		return catalog.PrintTemplateRepos(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(templatesCmd)
}
