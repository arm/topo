package main

import (
	"fmt"
	"os"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/arm-debug/topo-cli/internal/output/printable"
	"github.com/arm-debug/topo-cli/internal/output/templates"
	"github.com/spf13/cobra"
)

var templateFilters catalog.TemplateFilters

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}

		resolvedTarget, exists := lookupTarget(cmd)
		if exists {
			templateFilters.Target = resolvedTarget
		}

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		repos, err = catalog.FilterTemplateRepos(templateFilters, repos)
		if err != nil {
			return fmt.Errorf("could not filter templates: %w", err)
		}
		return printable.Print(templates.RepoCollection(repos), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(templatesCmd)
	templatesCmd.Flags().StringSliceVar(
		&templateFilters.Features,
		"feature",
		[]string{},
		"Only show templates that use the indicated arm feature (NEON, SVE, SME, SVE2, SME2)",
	)
	templatesCmd.MarkFlagsMutuallyExclusive("target", "feature")
	rootCmd.AddCommand(templatesCmd)
}
