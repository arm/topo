package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

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

		var profile *probe.HardwareProfile
		if targetArg, exists := lookupTarget(cmd); exists {
			r := runner.For(ssh.NewDestination(targetArg), runner.SSHOptions{Multiplex: true})
			ctx, cancel := contextWithTimeout(cmd)
			defer cancel()
			hwProfile, err := probe.Hardware(ctx, r)
			if err != nil {
				if errors.Is(err, ssh.ErrSSH) || errors.Is(err, runner.ErrTimeout) {
					return fmt.Errorf("failed to reach target %s: %w", targetArg, err)
				}
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
	rootCmd.AddCommand(templatesCmd)
}
