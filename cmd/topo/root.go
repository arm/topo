package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/arm-debug/topo-cli/internal/output/console"
	"github.com/arm-debug/topo-cli/internal/output/term"
	"github.com/arm-debug/topo-cli/internal/version"
	"github.com/spf13/cobra"
)

var output string

var rootCmd = &cobra.Command{
	Use:           "topo",
	Short:         "Topo CLI",
	Version:       fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&output,
		"output",
		"o",
		"plain",
		"Output format: plain or json",
	)
}

func addTargetFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "target", "", "The SSH destination.")
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

func GetLogger(cmd *cobra.Command) (*console.Logger, error) {
	flagValue, err := cmd.Flags().GetString("output")
	if err != nil {
		return nil, err
	}

	format, err := resolveOutput(flagValue)
	if err != nil {
		return nil, err
	}

	log := console.NewLogger(os.Stderr, format)
	return log, nil
}
