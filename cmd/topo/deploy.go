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
	checks "github.com/arm-debug/topo-cli/internal/deploy/project_checks"
	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/arm-debug/topo-cli/internal/ssh"

	"github.com/spf13/cobra"
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

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			panic(fmt.Sprintf("internal error: dry-run flag not registered: %v", err))
		}

		skipProjectChecks, err := cmd.Flags().GetBool("skip-project-checks")
		if err != nil {
			panic(fmt.Sprintf("internal error: no-registry flag not registered: %v", err))
		}

		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}
		c := console.NewLogger(os.Stderr, outputFormat)
		if err != nil {
			return err
		}

		deployOpts, err := lookupDeployOpts(cmd)
		if err != nil {
			return err
		}

		portChanged := cmd.Flags().Changed("port")
		if portChanged && !deployOpts.WithRegistry {
			c.Log(logger.Entry{
				Level:   logger.Warning,
				Message: "--port has no effect when --no-registry is set. Define SSH port in your SSH config instead.",
			})
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		if err := validatePort(deployOpts.Port); err != nil {
			return err
		}

		if !skipProjectChecks {
			if err := checks.EnsureProjectIsLinuxArm64Ready(composeFile); err != nil {
				return err
			}
		}

		if !deployOpts.WithRegistry {
			c.Log(logger.Entry{
				Level:   logger.Warning,
				Message: "registry transfer is not yet supported with this configuration. Falling back to direct transfer.",
			})
		}

		deployment, cleanup := docker.NewDeployment(composeFile, deployOpts)
		stop := goperation.SetupExitCleanup(os.Stdout, cleanup, os.Exit)

		defer func() {
			entries := stop()
			c.Log(entries...)
		}()

		if dryRun {
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

func lookupPort(cmd *cobra.Command) string {
	port, err := cmd.Flags().GetString("port")
	if err != nil {
		panic(fmt.Sprintf("internal error: port flag not registered: %v", err))
	}

	if env, ok := os.LookupEnv("TOPO_PORT"); ok && strings.TrimSpace(env) != "" {
		port = strings.TrimSpace(env)
	}

	return port
}

func lookupDeployOpts(cmd *cobra.Command) (docker.DeployOptions, error) {
	var deployOpts docker.DeployOptions

	targetHost, err := requireTarget(cmd)
	if err != nil {
		return deployOpts, err
	}
	deployOpts.TargetHost = ssh.Host(targetHost)

	noRegistry, err := cmd.Flags().GetBool("no-registry")
	if err != nil {
		panic(fmt.Sprintf("internal error: no-registry flag not registered: %v", err))
	}
	deployOpts.WithRegistry = docker.SupportsRegistry(noRegistry, deployOpts.TargetHost)

	deployOpts.Port = lookupPort(cmd)

	deployOpts.ForceRecreate, err = cmd.Flags().GetBool("force-recreate")
	if err != nil {
		panic(fmt.Sprintf("internal error: force-recreate flag not registered: %v", err))
	}
	deployOpts.NoRecreate, err = cmd.Flags().GetBool("no-recreate")
	if err != nil {
		panic(fmt.Sprintf("internal error: force-recreate flag not registered: %v", err))
	}

	goos := runtime.GOOS
	deployOpts.UseSSHControlSockets = docker.SupportsSSHControlSockets(goos)

	return deployOpts, nil
}

func init() {
	addTargetFlag(deployCmd)
	addDryRunFlag(deployCmd)
	deployCmd.Flags().StringP("port", "p", operation.DefaultRegistryPort, "Registry and SSH tunnel port (can also be set via TOPO_PORT env var)")
	deployCmd.Flags().Bool("no-registry", false, "Disable private registry flow; use direct save/load transfer")
	deployCmd.Flags().Bool("skip-project-checks", false, "Skip project compatibility checks for the target platform")
	deployCmd.Flags().Bool("force-recreate", false, "Force recreation of containers even if their configuration and image haven't changed")
	deployCmd.Flags().Bool("no-recreate", false, "Prevent recreation of containers even if their configuration and image have changed")
	deployCmd.MarkFlagsMutuallyExclusive("force-recreate", "no-recreate")
	rootCmd.AddCommand(deployCmd)
}
