package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

const sourceFlag = "source"

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"templates"},
	Short:   "List available Topo Projects",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()

		var projects []catalog.Project
		var err error
		source := getSource(cmd)
		switch source {
		case builtinProjects:
			projects, err = catalog.ListBuiltinProjects()
		default:
			projects, err = catalog.ListProjectsFromURL(ctx, source)
		}
		if err != nil {
			return err
		}

		var profile *probe.HardwareProfile
		if targetArg, exists := lookupTarget(cmd); exists {
			r := runner.For(ssh.NewDestination(targetArg))
			hwProfile, err := probe.Hardware(ctx, r)
			if err != nil {
				return err
			}
			profile = &hwProfile
		}

		projectsWithCompatibility := catalog.AnnotateCompatibility(profile, projects)
		return views.Print(views.ProjectList(projectsWithCompatibility), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(projectsCmd)
	addTimeoutFlag(projectsCmd, defaultTimeout)
	if experimentalFeaturesEnabled() {
		projectsCmd.Flags().StringP(sourceFlag, "s", "", "where to source projects' data from")
	}
	rootCmd.AddCommand(projectsCmd)
}

const builtinProjects = "builtin"

func getSource(cmd *cobra.Command) string {
	if experimentalFeaturesEnabled() {
		flagValue, err := cmd.Flags().GetString(sourceFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", sourceFlag, err))
		}
		if flagValue != "" {
			return flagValue
		}
	}
	return builtinProjects
}
