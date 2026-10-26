package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"

	"github.com/spf13/cobra"
)

var topoStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a currently running deployment",
	Long: `Stop services that are already running on the target using definitions in the compose file.

Executing this command does not remove the containers.

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Use -f to specify a different compose file.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		engine, err := getEngineSelection(cmd)
		if err != nil {
			return err
		}
		target, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		composeFilePath, err := resolveComposeFilePath(cmd)
		if err != nil {
			return err
		}
		envFiles, err := getEnvFiles(cmd, composeFilePath)
		if err != nil {
			return err
		}
		scope, err := project.BuildScope(composeFilePath, target.value, envFiles)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(target.value)
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return deploy.Stop(ctx, os.Stdout, scope, dest, deploy.EngineKind(engine.value))
	},
}

func init() {
	addTargetFlag(topoStopCmd)
	addComposeFileFlag(topoStopCmd)
	addEnvFileFlag(topoStopCmd)
	addEngineFlag(topoStopCmd)
	rootCmd.AddCommand(topoStopCmd)
}
