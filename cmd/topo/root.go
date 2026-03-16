package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/describe"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/version"
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

const targetEnvVar = "TOPO_TARGET"

func addTargetFlag(cmd *cobra.Command) {
	cmd.Flags().StringP(
		"target", "t", "",
		fmt.Sprintf("The SSH destination (can also be set via %s env var).", targetEnvVar),
	)
}

func addTargetDescriptionFlag(cmd *cobra.Command) {
	cmd.Flags().String(
		"target-description",
		"",
		"Path to the target description file used to show template compatibility",
	)
}

func addTargetDescriptionFlags(cmd *cobra.Command) {
	addTargetFlag(cmd)
	addTargetDescriptionFlag(cmd)
	cmd.MarkFlagsMutuallyExclusive("target", "target-description")
}

func addDryRunFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("dry-run", false, "Show what commands would be executed without actually running them")
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

func retrieveTargetDescription(cmd *cobra.Command) (*target.HardwareProfile, error) {
	targetDescriptionPath, err := cmd.Flags().GetString("target-description")
	if err != nil {
		panic(fmt.Sprintf("internal error: target-description flag not registered: %v", err))
	}

	if strings.TrimSpace(targetDescriptionPath) != "" {
		return describe.ReadTargetDescriptionFromFile(targetDescriptionPath)
	}

	resolvedTarget, exists := lookupTarget(cmd)
	if !exists {
		return nil, nil
	}

	conn := target.NewConnection(resolvedTarget, target.ConnectionOptions{Multiplex: true})
	hwProfile, err := describe.GenerateTargetDescription(conn)
	if err != nil {
		return nil, err
	}
	return &hwProfile, nil
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
