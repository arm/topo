package main

import (
	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var checkHealthTarget string

var checkHealthCmd = &cobra.Command{
	Use:   "check-health",
	Short: "Show information about the target and check the host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolved, err := resolveTarget(checkHealthTarget)
		if err != nil {
			return err
		}
		return core.CheckHealth(resolved)
	},
}

func init() {
	addTargetFlag(checkHealthCmd, &checkHealthTarget)
	rootCmd.AddCommand(checkHealthCmd)
}
