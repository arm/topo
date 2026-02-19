package main

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"

	"github.com/spf13/cobra"
)

var topoStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a currently running deployment",
	Long: `Stop services that are already running on the target host using definitions in the compose file.

Executing this command does not remove the containers.

The compose file (compose.yaml) must be in the current working directory, as this is used to select the containers to be stopped.

Use --dry-run to see what commands would be executed without actually running them.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			panic(fmt.Sprintf("internal error: dry-run flag not registered: %v", err))
		}

		resolvedTarget, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		targetHost := ssh.Host(resolvedTarget)

		stop := docker.NewDeploymentStop(composeFile, targetHost)
		if dryRun {
			return stop.DryRun(os.Stdout)
		}

		return stop.Run(os.Stdout)
	},
}

func init() {
	addTargetFlag(topoStopCmd)
	addDryRunFlag(topoStopCmd)
	rootCmd.AddCommand(topoStopCmd)
}
