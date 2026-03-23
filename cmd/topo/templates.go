package main

import (
	"os"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
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
		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}

		repos, err := catalog.ParseRepos(catalog.TemplatesJSON)
		if err != nil {
			return err
		}

		var profile *target.HardwareProfile
		if targetDescriptionPath != "" {
			profile, err = describe.ReadTargetDescriptionFromFile(targetDescriptionPath)
			if err != nil {
				return err
			}
		} else {
			targetArg, exists := lookupTarget(cmd)
			if exists {
				dest := ssh.NewDestination(targetArg)
				conn := target.NewConnection(dest, target.ConnectionOptions{Multiplex: true, ConnectTimeout: sshConnectTimeout})
				probe := target.NewHardwareProbe(&conn)
				hwProfile, err := probe.Probe()
				if err != nil {
					return err
				}
				profile = &hwProfile
			}
		}

		reposWithCompatibility := catalog.AnnotateCompatibility(profile, repos)
		return printable.Print(templates.RepoCollection(reposWithCompatibility), os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(templatesCmd)
	templatesCmd.Flags().StringVar(
		&targetDescriptionPath,
		"target-description",
		"",
		"Path to the target description file used to show template compatibility",
	)
	templatesCmd.MarkFlagsMutuallyExclusive("target", "target-description")
	rootCmd.AddCommand(templatesCmd)
}
