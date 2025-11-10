package main

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "topo",
	Short:   "Topo CLI",
	Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
}

func addTargetFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "target", "", "The SSH destination.")
}
