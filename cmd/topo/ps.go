package main

import (
	"os"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"

	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List services in a currently running deployment",
	Long: `List services that are already running on the target host using definitions in the compose file.

Executing this command does not remove the containers.

The compose file (compose.yaml) must be in the current working directory, as this is used to select the containers to be stopped.

Use --dry-run to see what commands would be executed without actually running them.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)

		ps := docker.NewDeploymentPs(composeFile, dest)

		return ps.Run(os.Stdout)
	},
}

func init() {
	addTargetFlag(topoPsCmd)
	rootCmd.AddCommand(topoPsCmd)
}
