package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/deploy/docker"
	"github.com/arm-debug/topo-cli/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	stopTarget string
	stopDryRun bool
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

		resolvedTarget, err := resolveTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		targetHost := ssh.Host(resolvedTarget)

		stop := docker.NewDeploymentStop(composeFile, targetHost)
		if stopDryRun {
			return stop.DryRun(os.Stdout)
		}

		return stop.Run(os.Stdout)
	},
}

func init() {
	addTargetFlag(topoStopCmd)
	topoStopCmd.Flags().BoolVar(&stopDryRun, "dry-run", false, "Show what commands would be executed without actually running them")
	rootCmd.AddCommand(topoStopCmd)
}
