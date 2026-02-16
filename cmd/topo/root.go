package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/arm-debug/topo-cli/internal/output/term"
	"github.com/arm-debug/topo-cli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "topo",
	Short:         "Topo CLI",
	Version:       fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().StringP(
		"output",
		"o",
		"plain",
		"Output format: plain or json",
	)
}

func addTargetFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("target", "t", "", "The SSH destination.")
}

func resolveTarget(cmd *cobra.Command) (string, error) {
	flagValue, err := cmd.Flags().GetString("target")
	if err != nil {
		panic(fmt.Sprintf("resolveTarget: flag not registered: %v", err))
	}

	const targetEnvVar = "TOPO_TARGET"

	if strings.TrimSpace(flagValue) != "" {
		return flagValue, nil
	}

	if env := strings.TrimSpace(os.Getenv(targetEnvVar)); env != "" {
		return env, nil
	}
	return "", fmt.Errorf("target not specified: provide --target or set TOPO_TARGET env var")
}

func resolveOutput(cmd *cobra.Command) (term.Format, error) {
	flagValue, err := cmd.Flags().GetString("output")
	if err != nil {
		return term.Plain, nil
	}

	v := strings.TrimSpace(strings.ToLower(flagValue))
	switch v {
	case "plain":
		return term.Plain, nil
	case "json":
		return term.JSON, nil
	default:
		err := fmt.Errorf("invalid output value %q: must be 'plain' or 'json'", flagValue)
		return term.Plain, err
	}
}
