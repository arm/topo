package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/arm-debug/topo-cli/internal/output"
	"github.com/spf13/cobra"
)

var templatesOutput string

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat, err := resolveOutput(templatesOutput)
		if err != nil {
			return err
		}
		printer := output.NewPrinter(os.Stdout, outputFormat)
		return catalog.PrintTemplateRepos(printer, catalog.TemplatesJSON)
	},
}

func init() {
	addOutputFlag(templatesCmd, &templatesOutput)
	rootCmd.AddCommand(templatesCmd)
}
