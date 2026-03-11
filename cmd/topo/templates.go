package main

import (
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/spf13/cobra"
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

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		profile, err := retrieveTargetDescription(cmd)
		if err != nil {
			return err
		}

		reposWithCompatibility := catalog.AnnotateCompatibility(profile, repos)
		return printable.Print(templates.RepoCollection(reposWithCompatibility), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetDescriptionFlags(templatesCmd)
	rootCmd.AddCommand(templatesCmd)
}
