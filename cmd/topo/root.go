package main

import (
	"context"
	"fmt"
	"os"
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
	Use:           "topo",
	Short:         "Topo CLI",
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

func getComposeFileName(cmd *cobra.Command) (string, error) {
	flag := cmd.Flag(composeFileFlag)
	if flag == nil {
		panic(fmt.Sprintf("internal error: compose file flag not registered: %s", composeFileFlag))
	}

	if flag.Changed {
		composeFile := strings.TrimSpace(flag.Value.String())
		if composeFile == "" {
			return "", fmt.Errorf("compose file path must not be empty")
		}
		return compose.RequireFile(composeFile)
	}

	return compose.FindDefaultFile()
}

const targetEnvVar = "TOPO_TARGET"

func addTargetFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(
		"target", "t", "",
		fmt.Sprintf("SSH destination (can also be set via %s env var)", targetEnvVar),
	)
}

func lookupTarget(cmd *cobra.Command) (string, bool) {
	flagValue, err := cmd.Flags().GetString("target")
	if err != nil {
		panic(fmt.Sprintf("internal error: target flag not registered: %v", err))
	}

	if strings.TrimSpace(flagValue) == "" {
		flagValue = os.Getenv(targetEnvVar)
	}

	v := strings.TrimSpace(flagValue)
	if v == "" {
		return "", false
	}

	return v, true
}

func requireTarget(cmd *cobra.Command) (string, error) {
	t, exists := lookupTarget(cmd)
	if !exists {
		return "", fmt.Errorf("target not specified: provide --target or set TOPO_TARGET env var")
	}
	return t, nil
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

func experimentalFeaturesEnabled() bool {
	const experimentalFeaturesEnvVar = "TOPO_EXPERIMENTAL_FEATURES"
	return env.IsVarTruthy(experimentalFeaturesEnvVar)
}
