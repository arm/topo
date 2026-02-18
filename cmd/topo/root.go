package main

import (
	"fmt"
	"os"
	"regexp"
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

var sshTargetRgx = regexp.MustCompile(
	`^(?:[a-zA-Z0-9._-]+@)?[a-zA-Z0-9._-]+(?::\d+)?$`,
)

func lookupTarget(cmd *cobra.Command) (string, bool, error) {
	const targetEnvVar = "TOPO_TARGET"

	flagValue, err := cmd.Flags().GetString("target")
	if err != nil {
		panic(fmt.Sprintf("internal error: target flag not registered: %v", err))
	}

	if strings.TrimSpace(flagValue) == "" {
		flagValue = os.Getenv(targetEnvVar)
	}

	v := strings.TrimSpace(flagValue)
	if v == "" {
		return "", false, nil
	}

	if !sshTargetRgx.MatchString(v) {
		return "", false, fmt.Errorf("invalid SSH target: %q", v)
	}

	return v, true, nil
}

func requireTarget(cmd *cobra.Command) (string, error) {
	t, exists, err := lookupTarget(cmd)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("target not specified: provide --target or set TOPO_TARGET env var")
	}
	return t, nil
}

func resolveOutput(cmd *cobra.Command) (term.Format, error) {
	flagValue, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(fmt.Sprintf("internal error: output flag not registered: %v", err))
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
