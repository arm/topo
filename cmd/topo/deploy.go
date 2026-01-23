package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/arm-debug/topo-cli/internal/deploy/docker"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	deployTarget  string
	deployDryRun  bool
	forceRecreate bool
	noRegistry    bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy services using the compose file",
	Long: `Deploy services to the target host using definitions in the compose file.

This command performs the following operations in sequence:
  1. Build - Builds Container images defined in the compose file on the local host
  2. Pull - Pulls any required images from registries to the local host
  3. Transfer - Transfers built and pulled images and compose file to the target host
  4. Run - Runs docker compose up on the target host

The compose file (compose.yaml) must be in the current working directory, as this is used to select the containers to be deployed.

Use --dry-run to see what commands would be executed without actually running them.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		resolvedTarget, err := resolveTarget(deployTarget)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		targetHost := ssh.Host(resolvedTarget)
		goos := runtime.GOOS
		useRegistry := docker.SupportsRegistry(noRegistry, targetHost, goos)

		if !useRegistry {
			_, _ = fmt.Fprintln(os.Stderr, "WARN: Registry transfer is not yet supported with this configuration. Falling back to direct transfer.")
		}

		var deployment goperation.Sequence
		if useRegistry {
			deployment = docker.NewDeploymentWithRegistry(composeFile, targetHost, forceRecreate)
		} else {
			deployment = docker.NewDeployment(composeFile, targetHost, forceRecreate)
		}

		if deployDryRun {
			return deployment.DryRun(os.Stdout)
		}

		if useRegistry {
			cancel, cleanup := ssh.SetupTunnelCleanup(targetHost, os.Stdout)
			defer cleanup()
			defer cancel()
		}
		return deployment.Run(os.Stdout)
	},
}

func getComposeFileName() (string, error) {
	candidates := []string{"compose.yaml", "compose.yml"}
	for _, fileName := range candidates {
		if _, err := os.Stat(fileName); err == nil {
			return fileName, nil
		}
	}
	return "", fmt.Errorf("compose file not found in current working directory: looking for compose.yaml or compose.yml")
}

func init() {
	addTargetFlag(deployCmd, &deployTarget)
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Show what commands would be executed without actually running them")
	deployCmd.Flags().BoolVar(&noRegistry, "no-registry", false, "Disable private registry flow; use direct save/load transfer")
	deployCmd.Flags().BoolVar(&forceRecreate, "force-recreate", false, "Force recreation of containers even if their configuration and image haven't changed")
	rootCmd.AddCommand(deployCmd)
}
