package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/arm-debug/topo-cli/internal/deploy/docker"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	deployTarget string
	deployDryRun bool
	noRegistry   bool
	port         string
)

var deployOpts docker.DeployOptions

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

		portChanged := cmd.Flags().Changed("port")
		if portChanged && noRegistry {
			_, _ = fmt.Fprintln(os.Stderr, "WARN: --port has no effect when --no-registry is set. Define SSH port in your SSH config instead.")
		}

		resolvedTarget, err := resolveTarget(deployTarget)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		resolvedPort, err := resolvePort(cmd, port)
		if err != nil {
			return err
		}

		if err := validatePort(resolvedPort); err != nil {
			return err
		}

		targetHost := ssh.Host(resolvedTarget)
		deployOpts.TargetHost = targetHost
		deployOpts.Port = resolvedPort

		goos := runtime.GOOS
		deployOpts.WithRegistry = docker.SupportsRegistry(noRegistry, targetHost)
		deployOpts.UseSSHControlSockets = docker.SupportsSSHControlSockets(goos)

		if !deployOpts.WithRegistry {
			_, _ = fmt.Fprintln(os.Stderr, "WARN: Registry transfer is not yet supported with this configuration. Falling back to direct transfer.")
		}

		deployment, cleanup := docker.NewDeployment(composeFile, deployOpts)
		stop := goperation.SetupExitCleanup(cleanup, os.Stderr, os.Exit)
		defer stop()

		if deployDryRun {
			return deployment.DryRun(os.Stdout)
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

func validatePort(port string) error {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port %q: must be a number", port)
	}
	if portNum < 1 || portNum > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", portNum)
	}
	return nil
}

func resolvePort(cmd *cobra.Command, flagValue string) (string, error) {
	const portEnvVar = "TOPO_PORT"

	if cmd.Flags().Changed("port") {
		return flagValue, nil
	}
	if env := strings.TrimSpace(os.Getenv(portEnvVar)); env != "" {
		return env, nil
	}
	return flagValue, nil
}

func init() {
	addTargetFlag(deployCmd, &deployTarget)
	deployCmd.Flags().StringVarP(&port, "port", "p", operation.DefaultRegistryPort, "Registry and SSH tunnel port (can also be set via TOPO_PORT env var)")
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "Show what commands would be executed without actually running them")
	deployCmd.Flags().BoolVar(&noRegistry, "no-registry", false, "Disable private registry flow; use direct save/load transfer")
	deployCmd.Flags().BoolVar(&deployOpts.ForceRecreate, "force-recreate", false, "Force recreation of containers even if their configuration and image haven't changed")
	deployCmd.Flags().BoolVar(&deployOpts.NoRecreate, "no-recreate", false, "Prevent recreation of containers even if their configuration and image have changed")
	deployCmd.MarkFlagsMutuallyExclusive("force-recreate", "no-recreate")
	rootCmd.AddCommand(deployCmd)
}
