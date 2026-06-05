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

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "List available Topo Templates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		ctx, cancel := contextWithTimeout(cmd)
		defer cancel()

		var templates []catalog.Template
		var err error
		source := getSource(cmd)
		switch source {
		case builtinTemplates:
			templates, err = catalog.ListBuiltinTemplates()
		default:
			templates, err = catalog.ListTemplatesFromURL(ctx, source)
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

		templatesWithCompatibility := catalog.AnnotateCompatibility(profile, templates)
		return views.Print(views.TemplateList(templatesWithCompatibility), os.Stdout, outputFormat)
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

const builtinTemplates = "builtin"

func getSource(cmd *cobra.Command) string {
	if experimentalFeaturesEnabled() {
		flagValue, err := cmd.Flags().GetString(sourceFlag)
		if err != nil {
			panic(fmt.Sprintf("internal error: %s flag not registered: %v", sourceFlag, err))
		}
		return flagValue
	}
	return builtinTemplates
}
