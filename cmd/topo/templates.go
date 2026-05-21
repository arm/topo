package main

import (
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

const sourceFlag = "source"

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Topo Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		var repos []catalog.Repo
		var err error
		source := getSource(cmd)
		switch source {
		case "":
			repos, err = catalog.ListBuiltinTemplates()
		default:
			repos, err = catalog.ListTemplatesFromURL(source)
		}
		if err != nil {
			return err
		}

		var profile *probe.HardwareProfile
		if targetArg, exists := lookupTarget(cmd); exists {
			r := runner.For(ssh.NewDestination(targetArg))
			ctx, cancel := contextWithTimeout(cmd)
			defer cancel()
			hwProfile, err := probe.Hardware(ctx, r)
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
	if experimentalFeaturesEnabled() {
		templatesCmd.Flags().StringP(sourceFlag, "s", "", "where to source templates' data from")
	}
	rootCmd.AddCommand(templatesCmd)
}

func getSource(cmd *cobra.Command) string {
	if experimentalFeaturesEnabled() {
		flagValue, err := cmd.Flags().GetString("source")
		if err != nil {
			panic(fmt.Sprintf("internal error: source flag not registered: %v", err))
		}
		return flagValue
	}
	return ""
}
