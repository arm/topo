package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
		"Output format: plain or json",
	)
}

const targetEnvVar = "TOPO_TARGET"

func addTargetFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(
		"target", "t", "",
		fmt.Sprintf("The SSH destination (can also be set via %s env var).", targetEnvVar),
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
	cmd.Flags().Duration("timeout", defaultTimeout, "Maximum time to wait for the command to complete (0 to disable timeout)")
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
