package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/docker/operation"
	checks "github.com/arm/topo/internal/deploy/project_checks"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	noRegistry        bool
	registryPort      string
	skipProjectChecks bool
	forceRecreate     bool
	noRecreate        bool
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

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			panic(fmt.Sprintf("internal error: dry-run flag not registered: %v", err))
		}

		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}
		c := console.NewLogger(os.Stderr, outputFormat)
		if err != nil {
			return err
		}

		portChanged := cmd.Flags().Changed("registry-port")
		if portChanged && noRegistry {
			c.Log(logger.Entry{
				Level:   logger.Warning,
				Message: "--registry-port has no effect when --no-registry is set. Define SSH port in your SSH config instead.",
			})
		}

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		resolvedPort, err := resolvePort(cmd, registryPort)
		if err != nil {
			return err
		}

		if err := validatePort(resolvedPort); err != nil {
			return err
		}

		deployOpts.TargetHost = ssh.NewDestination(targetArg)

		if !skipProjectChecks {
			if err := checks.EnsureProjectIsLinuxArm64Ready(composeFile); err != nil {
				return err
			}
		}

		goos := runtime.GOOS
		if docker.SupportsRegistry(noRegistry, deployOpts.TargetHost) {
			deployOpts.Registry = &docker.RegistryConfig{
				Port:              resolvedPort,
				UseControlSockets: docker.SupportsSSHControlSockets(goos),
			}
		}
		switch {
		case forceRecreate:
			deployOpts.RecreateMode = operation.RecreateModeForce
		case noRecreate:
			deployOpts.RecreateMode = operation.RecreateModeNone
		}

		if deployOpts.Registry == nil {
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

const portEnvVar = "TOPO_PORT"

func resolvePort(cmd *cobra.Command, flagValue string) (string, error) {
	if cmd.Flags().Changed("registry-port") {
		return flagValue, nil
	}
	if env := strings.TrimSpace(os.Getenv(portEnvVar)); env != "" {
		return env, nil
	}
	return flagValue, nil
}

func init() {
	addTargetFlag(deployCmd)
	addDryRunFlag(deployCmd)
	deployCmd.Flags().StringVarP(&registryPort, "registry-port", "p", operation.DefaultRegistryPort, fmt.Sprintf("Registry and SSH tunnel port (can also be set via %s env var)", portEnvVar))
	deployCmd.Flags().BoolVar(&noRegistry, "no-registry", false, "Disable private registry flow; use direct save/load transfer")
	deployCmd.Flags().BoolVar(&forceRecreate, "force-recreate", false, "Force recreation of containers even if their configuration and image haven't changed")
	deployCmd.Flags().BoolVar(&noRecreate, "no-recreate", false, "Prevent recreation of containers even if their configuration and image have changed")
	deployCmd.Flags().BoolVar(&skipProjectChecks, "skip-project-checks", false, "Skip project compatibility checks for the target platform")
	deployCmd.MarkFlagsMutuallyExclusive("force-recreate", "no-recreate")
	rootCmd.AddCommand(deployCmd)
}
