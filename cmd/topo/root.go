package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "topo",
	Short: "Discover and deploy containerised software to Arm hardware over SSH",
	Long: `Topo discovers and deploys containerised software to Arm hardware over SSH.

Use Topo to find hardware-matched Topo Projects, configure them, and deploy
them as standard Docker Compose projects. You can also deploy existing Compose
projects with fast, incremental builds.

Topo runs on a host and deploys applications to a target.

The host is the system where the Topo CLI runs. The target is the Linux/Arm64
system where deployments run, reached over SSH.

Commands requiring a target accept an SSH destination (for example,
user@example.local) with --target  or the TOPO_TARGET environment variable.

The host and target may be the same system; use --target localhost in that
case.`,
	Version:       fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		outputFormat := resolveOutput(cmd)
		logger.SetOptions(logger.Options{Format: outputFormat})
	},
}

func init() {
	rootCmd.PersistentFlags().StringP(
		"output",
		"o",
		"plain",
		"output format: plain or json",
	)
}

const composeFileFlag = "file"

func addComposeFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(
		composeFileFlag,
		"f",
		"",
		"compose file to use (default: compose.yaml, then compose.yml)",
	)
}

func resolveComposeFilePath(cmd *cobra.Command) (string, error) {
	flag := cmd.Flag(composeFileFlag)
	if flag == nil {
		panic(fmt.Sprintf("internal error: compose file flag not registered: %s", composeFileFlag))
	}

	if flag.Changed {
		composeFileArg := strings.TrimSpace(flag.Value.String())
		if composeFileArg == "" {
			return "", fmt.Errorf("compose file path must not be empty")
		}
		composeFilePath, err := filepath.Abs(composeFileArg)
		if err != nil {
			return "", fmt.Errorf("failed to resolve compose file path: %w", err)
		}
		return compose.RequireFile(composeFilePath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	return compose.FindDefaultFile(cwd)
}

const targetEnvVar = env.TargetVariable

type containerEngine string

type engineSelection struct {
	value    containerEngine
	explicit bool
}

const (
	containerEngineDocker containerEngine = "docker"
	containerEnginePodman containerEngine = "podman"
)

const (
	containerEngineFlag   = "engine"
	containerEngineEnvVar = "TOPO_ENGINE"
)

func addEngineFlag(cmd *cobra.Command) {
	cmd.Flags().String(
		containerEngineFlag,
		string(containerEngineDocker),
		fmt.Sprintf("container engine to use (docker or podman; can also be set via %s)", containerEngineEnvVar),
	)
}

func getEngineSelection(cmd *cobra.Command) (engineSelection, error) {
	if cmd.Flags().Lookup(containerEngineFlag) == nil {
		return engineSelection{value: containerEngineDocker}, nil
	}

	value, err := cmd.Flags().GetString(containerEngineFlag)
	if err != nil {
		panic(fmt.Sprintf("internal error: container engine flag is not a string: %v", err))
	}

	explicit := cmd.Flags().Changed(containerEngineFlag)
	if !explicit {
		if envValue := strings.TrimSpace(os.Getenv(containerEngineEnvVar)); envValue != "" {
			value = envValue
		}
	}

	selectedEngine := containerEngine(value)
	if selectedEngine != containerEngineDocker && selectedEngine != containerEnginePodman {
		return engineSelection{}, fmt.Errorf("invalid engine %q: must be docker or podman", value)
	}
	return engineSelection{value: selectedEngine, explicit: explicit}, nil
}

func addTargetFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(
		"target", "t", "",
		fmt.Sprintf("SSH destination (can also be set via %s env var)", targetEnvVar),
	)
}

type targetSelection struct {
	value    string
	explicit bool
}

func lookupTarget(cmd *cobra.Command) (targetSelection, bool) {
	flagValue, err := cmd.Flags().GetString("target")
	if err != nil {
		panic(fmt.Sprintf("internal error: target flag not registered: %v", err))
	}

	if value := strings.TrimSpace(flagValue); value != "" {
		return targetSelection{value: value, explicit: cmd.Flags().Changed("target")}, true
	}

	value := strings.TrimSpace(os.Getenv(targetEnvVar))
	if value == "" {
		return targetSelection{}, false
	}
	return targetSelection{value: value}, true
}

func requireTarget(cmd *cobra.Command) (targetSelection, error) {
	target, exists := lookupTarget(cmd)
	if !exists {
		return targetSelection{}, fmt.Errorf("target not specified: use --target with an SSH destination (e.g. user@example.local) or set %s", targetEnvVar)
	}
	return target, nil
}

const defaultTimeout = 5 * time.Second

func addTimeoutFlag(cmd *cobra.Command, defaultTimeout time.Duration) {
	cmd.Flags().Duration("timeout", defaultTimeout, "maximum time to wait for the command to complete (0 to disable timeout)")
}

func contextWithTimeout(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		panic(fmt.Sprintf("internal error: timeout flag not registered: %v", err))
	}
	if timeout == 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), timeout)
}

func resolveOutput(cmd *cobra.Command) term.Format {
	flagValue, _ := cmd.Flags().GetString("output")
	v := strings.TrimSpace(strings.ToLower(flagValue))
	if v == "json" {
		return term.JSON
	}
	return term.Plain
}

const envFileFlag = "env-file"

func addEnvFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringArray(
		envFileFlag, []string{".env", env.DefaultFilename},
		"path to env file to source values for compose interpolation",
	)
}

func getEnvFiles(cmd *cobra.Command, composeFilePath string) ([]string, error) {
	envFiles, err := cmd.Flags().GetStringArray(envFileFlag)
	if err != nil {
		panic(fmt.Sprintf("internal error: env-file flag not registered: %v", err))
	}

	root := filepath.Dir(composeFilePath)
	skipMissing := true
	if cmd.Flag(envFileFlag).Changed {
		skipMissing = false
		root, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
	}

	return env.ResolveFiles(root, envFiles, skipMissing)
}

const migrateToEnvFlag = "migrate-to-env"

func addMigrateToEnvFlag(cmd *cobra.Command) {
	cmd.Flags().Bool(migrateToEnvFlag, false, fmt.Sprintf("move parameter values from the compose file to %q, updating the compose file accordingly", env.DefaultFilename))
}

func migrateToEnv(cmd *cobra.Command) bool {
	if cmd.Flags().Lookup(migrateToEnvFlag) == nil {
		return false
	}
	enabled, err := cmd.Flags().GetBool(migrateToEnvFlag)
	if err != nil {
		panic(fmt.Sprintf("internal error: migrate-to-env flag not registered: %v", err))
	}
	return enabled
}

func experimentalFeaturesEnabled() bool {
	const experimentalFeaturesEnvVar = "TOPO_EXPERIMENTAL_FEATURES"
	return env.IsVarTruthy(experimentalFeaturesEnvVar)
}
