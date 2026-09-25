package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	cmdtext "github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	checks "github.com/arm/topo/internal/deploy/project_checks"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/project"
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

		if cmd.Flags().Changed("registry-port") && noRegistry {
			logger.Warn("--registry-port has no effect when --no-registry is set. Define a port in your ssh config instead.")
		}
		resolvedPort, err := resolveValidPort(cmd, registryPort)
		if err != nil {
			return err
		}

		if err := ensureProjectIsReady(scope); err != nil {
			return err
		}

		composeFileFlagValue := cmd.Flag(composeFileFlag)
		defaultSuccessMessage := buildDefaultSuccessMessage(
			engine,
			target,
			strings.TrimSpace(composeFileFlagValue.Value.String()),
			composeFileFlagValue.Changed,
		)

		options := deploy.Options{
			TargetHost:            ssh.NewDestination(target.value),
			DefaultSuccessMessage: defaultSuccessMessage,
		}
		if !noRegistry {
			options.Registry = &deploy.RegistryConfig{
				Port:                resolvedPort,
				SkipRemotePortCheck: resolveSkipRemotePortCheck(cmd),
			}
		}
		switch {
		case forceRecreate:
			options.RecreateMode = deploy.RecreateModeForce
		case noRecreate:
			options.RecreateMode = deploy.RecreateModeNone
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		var deploymentErr error
		if engine.value == containerEnginePodman {
			deploymentErr = podman.Deploy(ctx, os.Stdout, scope, options)
		} else {
			deploymentErr = docker.Deploy(ctx, os.Stdout, scope, options)
		}
		if deploymentErr == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		healthCommand := buildHealthCommand(engine, target)
		return fmt.Errorf("deployment failed; ensure `%s` is passing: %w", healthCommand, deploymentErr)
	},
}

func buildDefaultSuccessMessage(engine engineSelection, target targetSelection, composeFilePath string, explicitComposeFile bool) string {
	composeFileArg := ""
	if explicitComposeFile {
		composeFileArg = "-f " + cmdtext.QuoteArg(composeFilePath)
	}
	psCommand := joinNonEmpty("topo ps", engine.cliArg(), target.cliArg(), composeFileArg)
	return fmt.Sprintf("Run `%s` to see deployed containers", psCommand)
}

func buildHealthCommand(engine engineSelection, target targetSelection) string {
	return joinNonEmpty("topo health", engine.cliArg(), target.cliArg())
}

func (engine engineSelection) cliArg() string {
	if !engine.explicit {
		return ""
	}
	return "--engine " + string(engine.value)
}

func (target targetSelection) cliArg() string {
	if !target.explicit {
		return ""
	}
	return "--target " + cmdtext.QuoteArg(target.value)
}

func joinNonEmpty(args ...string) string {
	nonEmptyArgs := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != "" {
			nonEmptyArgs = append(nonEmptyArgs, arg)
		}
	}
	return strings.Join(nonEmptyArgs, " ")
}

func ensureProjectIsReady(scope project.Scope) error {
	if skipProjectChecks {
		return nil
	}
	return checks.EnsureProjectIsLinuxArm64Ready(scope)
}

const (
	portEnvVar                = "TOPO_PORT"
	skipRemotePortCheckEnvVar = "TOPO_SKIP_REMOTE_PORT_CHECK"
)

func resolveValidPort(cmd *cobra.Command, flagValue string) (string, error) {
	port := flagValue
	if !cmd.Flags().Changed("registry-port") {
		if envPort := strings.TrimSpace(os.Getenv(portEnvVar)); envPort != "" {
			port = envPort
		}
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("invalid port %q: must be a number", port)
	}
	if portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("invalid port %d: must be between 1 and 65535", portNumber)
	}
	return port, nil
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
	addEnvFileFlag(deployCmd)
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
