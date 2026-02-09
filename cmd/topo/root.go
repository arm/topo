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

func addTargetFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "target", "", "The SSH destination.")
}

func addOutputFlag(cmd *cobra.Command, output *string) {
	cmd.Flags().StringVarP(output, "output", "o", "plain", "Output format: plain or json")
}

func addLogOutputFlag(cmd *cobra.Command, output *string) {
	cmd.Flags().StringVarP(output, "log-output", "l", "plain", "Output format: plain or json")
}

func resolveTarget(flagValue string) (string, error) {
	const targetEnvVar = "TOPO_TARGET"

	if strings.TrimSpace(flagValue) != "" {
		return flagValue, nil
	}
	if env := strings.TrimSpace(os.Getenv(targetEnvVar)); env != "" {
		return env, nil
	}
	return "", fmt.Errorf("target not specified: provide --target or set TOPO_TARGET env var")
}

func resolveOutput(flagValue string) (term.Format, error) {
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
