package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/target"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	templateFilters       catalog.TemplateFilters
	targetDescriptionPath string
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}

		// even if the target flag was not used, TOPO_TARGET may be set, so we check if the resolved target is non-empty
		if targetDescriptionPath == "" {
			resolvedTarget, exists := lookupTarget(cmd)
			if exists {
				templateFilters.Target = resolvedTarget
			}
		}

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		repos, err = catalog.FilterTemplateRepos(templateFilters, repos)
		if err != nil {
			return fmt.Errorf("could not filter templates: %w", err)
		}

		reposWithCompatibility := catalog.WithCompatibility(repos)

		if targetDescriptionPath != "" {
			description, err := os.ReadFile(targetDescriptionPath)
			if err != nil {
				return fmt.Errorf("failed to read target description file %q: %w", targetDescriptionPath, err)
			}

			var profile target.HardwareProfile
			if err := yaml.Unmarshal(description, &profile); err != nil {
				return fmt.Errorf("failed to parse target description file %q: %w", targetDescriptionPath, err)
			}
			reposWithCompatibility = catalog.AnnotateCompatibility(profile, reposWithCompatibility)
		}

		return printable.Print(templates.RepoCollection(reposWithCompatibility), os.Stdout, outputFormat)
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
	templatesCmd.Flags().StringVar(
		&targetDescriptionPath,
		"target-description",
		"",
		"Path to the target description file used to show template compatibility",
	)
	templatesCmd.MarkFlagsMutuallyExclusive("target", "feature")
	templatesCmd.MarkFlagsMutuallyExclusive("target", "target-description")
	rootCmd.AddCommand(templatesCmd)
}
