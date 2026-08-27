package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	noRegistry          bool
	registryPort        string
	skipRemotePortCheck bool
	skipProjectChecks   bool
	forceRecreate       bool
	noRecreate          bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy services using the compose file",
	Long: `Deploy services to the target using definitions in the compose file.

This command performs the following operations in sequence:
  1. Build - Builds container images defined in the compose file on the host
  2. Pull - Pulls any required images from registries to the host
  3. Transfer - Transfers built and pulled images and compose file to the target
  4. Run - Runs docker compose up on the target

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Use -f to specify a different compose file.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		selectedEngine, err := getSelectedEngine(cmd)
		if err != nil {
			return err
		}
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		err = env.SetTargetEnv(targetArg)
		if err != nil {
			return err
		}
		if selectedEngine == containerEnginePodman {
			return deployWithPodman(cmd, targetArg)
		}
		return deployWithDocker(cmd, targetArg)
	},
}

func deployWithPodman(cmd *cobra.Command, targetArg string) error {
	if cmd.Flags().Changed("registry-port") && noRegistry {
		logger.Warn("--registry-port has no effect when --no-registry is set. Define a port in your ssh config instead.")
	}

	composeFile, err := getComposeFileName(cmd)
	if err != nil {
		return err
	}
	if err := ensureProjectIsReady(composeFile); err != nil {
		return err
	}

	resolvedPort, err := resolvePort(cmd, registryPort)
	if err != nil {
		return err
	}
	if err := validatePort(resolvedPort); err != nil {
		return err
	}

	targetHost := ssh.NewDestination(targetArg)
	options := podman.DeployOptions{TargetHost: targetHost}
	if !noRegistry {
		options.Registry = &podman.RegistryConfig{
			Port:                resolvedPort,
			SkipRemotePortCheck: resolveSkipRemotePortCheck(cmd),
		}
	}
	switch {
	case forceRecreate:
		options.RecreateMode = podman.RecreateModeForce
	case noRecreate:
		options.RecreateMode = podman.RecreateModeNone
	}

	return executeDeployment(cmd, func(ctx context.Context) error {
		return podman.Deploy(ctx, os.Stdout, composeFile, options)
	})
}

func deployWithDocker(cmd *cobra.Command, targetArg string) error {
	if cmd.Flags().Changed("registry-port") && noRegistry {
		logger.Warn("--registry-port has no effect when --no-registry is set. Define a port in your ssh config instead.")
	}

	composeFile, err := getComposeFileName(cmd)
	if err != nil {
		return err
	}
	if err := ensureProjectIsReady(composeFile); err != nil {
		return err
	}

	resolvedPort, err := resolvePort(cmd, registryPort)
	if err != nil {
		return err
	}
	if err := validatePort(resolvedPort); err != nil {
		return err
	}

	deployOpts := docker.DeployOptions{TargetHost: ssh.NewDestination(targetArg)}
	if !noRegistry {
		deployOpts.Registry = &docker.RegistryConfig{
			Port:                resolvedPort,
			SkipRemotePortCheck: resolveSkipRemotePortCheck(cmd),
		}
	}
	switch {
	case forceRecreate:
		deployOpts.RecreateMode = docker.RecreateModeForce
	case noRecreate:
		deployOpts.RecreateMode = docker.RecreateModeNone
	}

	return executeDeployment(cmd, func(ctx context.Context) error {
		return docker.Deploy(ctx, os.Stdout, composeFile, deployOpts)
	})
}

func ensureProjectIsReady(composeFile string) error {
	if skipProjectChecks {
		return nil
	}
	return checks.EnsureProjectIsLinuxArm64Ready(composeFile)
}

func executeDeployment(cmd *cobra.Command, deployment func(context.Context) error) error {
	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := deployment(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("deployment failed; ensure topo health is passing: %w", err)
	}
	return nil
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

const (
	portEnvVar                = "TOPO_PORT"
	skipRemotePortCheckEnvVar = "TOPO_SKIP_REMOTE_PORT_CHECK"
)

func resolvePort(cmd *cobra.Command, flagValue string) (string, error) {
	if cmd.Flags().Changed("registry-port") {
		return flagValue, nil
	}
	if env := strings.TrimSpace(os.Getenv(portEnvVar)); env != "" {
		return env, nil
	}
	return flagValue, nil
}

func resolveSkipRemotePortCheck(cmd *cobra.Command) bool {
	flagValue, _ := cmd.Flags().GetBool("skip-remote-port-check")
	if cmd.Flags().Changed("skip-remote-port-check") {
		return flagValue
	}

	return env.IsVarTruthy(skipRemotePortCheckEnvVar)
}

func init() {
	addTargetFlag(deployCmd)
	addComposeFileFlag(deployCmd)
	if experimentalFeaturesEnabled() {
		addEngineFlag(deployCmd)
	}
	deployCmd.Flags().StringVarP(&registryPort, "registry-port", "p", docker.DefaultRegistryPort, fmt.Sprintf("registry and SSH tunnel port (can also be set via %s env var)", portEnvVar))
	deployCmd.Flags().BoolVar(&noRegistry, "no-registry", false, "use full-image save/load transfer instead of delta-optimised registry transfer; for environments where registry transfer or reverse SSH forwarding is unavailable")
	deployCmd.Flags().BoolVar(&skipRemotePortCheck, "skip-remote-port-check", false, fmt.Sprintf("skip checking whether the SSH tunnel port is exposed on the remote network (can also be set via %s env var)", skipRemotePortCheckEnvVar))
	deployCmd.Flags().BoolVar(&forceRecreate, "force-recreate", false, "force recreation of containers even if their configuration and image haven't changed")
	deployCmd.Flags().BoolVar(&noRecreate, "no-recreate", false, "prevent recreation of containers even if their configuration and image have changed")
	deployCmd.Flags().BoolVar(&skipProjectChecks, "skip-project-checks", false, "skip project compatibility checks for the target platform")
	deployCmd.MarkFlagsMutuallyExclusive("force-recreate", "no-recreate")
	rootCmd.AddCommand(deployCmd)
}
