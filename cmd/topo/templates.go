package main

import (
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/spf13/cobra"
)

var targetDescriptionPath string

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Service Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		var profile *target.HardwareProfile
		if targetArg, exists := lookupTarget(cmd); exists {
			r := runner.For(ssh.NewDestination(targetArg), runner.SSHOptions{Multiplex: true})
			ctx, cancel := contextWithTimeout(cmd)
			defer cancel()
			hwProfile, err := target.ProbeHardware(ctx, r)
			if err != nil {
				return err
			}
			profile = &hwProfile
		}

		reposWithCompatibility := catalog.AnnotateCompatibility(profile, repos)
		return printable.Print(templates.RepoCollection(reposWithCompatibility), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(templatesCmd)
	addTimeoutFlag(templatesCmd, defaultTimeout)
	templatesCmd.Flags().StringVar(
		&targetDescriptionPath,
		"target-description",
		"",
		"Path to the target description file used to show template compatibility",
	)
	templatesCmd.MarkFlagsMutuallyExclusive("target", "target-description")
	rootCmd.AddCommand(templatesCmd)
}
