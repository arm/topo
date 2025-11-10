package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/spf13/cobra"
)

var listServiceTemplatesCmd = &cobra.Command{
	Use:   "list-service-templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return template.PrintList(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(listServiceTemplatesCmd)
}
