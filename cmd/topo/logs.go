package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var logsFollow bool

var topoLogsCmd = &cobra.Command{
	Use:   "logs [SERVICE...]",
	Short: "Show logs from deployed Compose services",
	Long: `Show logs from deployed Compose services, including stopped containers.
Omit service names to show logs from all services in the current project.
Use --follow to stream new log output until interrupted with Ctrl+C.

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Use -f to specify a different compose file.
Log output is streamed as text.`,
	Example: `  topo logs --target user@board
  topo logs --target user@board --follow web
  topo logs --target localhost --engine podman web worker`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, services []string) error {
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

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		dest := ssh.NewDestination(target.value)
		options := deploy.LogsOptions{Services: services, Follow: logsFollow}
		err = deploy.Logs(ctx, os.Stdout, scope, dest, deploy.EngineKind(engine.value), options)
		if ctx.Err() == context.Canceled {
			return nil
		}
		return err
	},
}

func init() {
	addTargetFlag(topoLogsCmd)
	addComposeFileFlag(topoLogsCmd)
	addEnvFileFlag(topoLogsCmd)
	addEngineFlag(topoLogsCmd)
	topoLogsCmd.Flags().BoolVar(&logsFollow, "follow", false, "follow new log output until interrupted")
	rootCmd.AddCommand(topoLogsCmd)
}
