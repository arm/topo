package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/arm-debug/topo-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	templateFilters catalog.TemplateFilters
	templatesOutput string
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat, err := resolveOutput(templatesOutput)
		if err != nil {
			return err
		}

		if templateFilters.Target != "" {
			target, err := resolveTarget(templateFilters.Target)
			if err != nil {
				return err
			}
			templateFilters.Target = target
		}

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		repos = catalog.FilterTemplateRepos(templateFilters, repos)
		printer := output.NewPrinter(os.Stdout, outputFormat)
		return output.PrintTemplateRepos(printer, repos)
	},
}

func init() {
	addTargetFlag(templatesCmd, &templateFilters.Target)
	addOutputFlag(templatesCmd, &templatesOutput)
	templatesCmd.Flags().StringSliceVar(
		&templateFilters.Features,
		"feature",
		[]string{},
		"Only show templates that use the indicated arm feature (NEON, SVE, SME, SVE2, SME2)",
	)
	templatesCmd.MarkFlagsMutuallyExclusive("target", "feature")
	rootCmd.AddCommand(templatesCmd)
}
