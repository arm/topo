package main

import (
	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/spf13/cobra"
)

var healthTarget string

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the target host environment (container engines, SSH availability)",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		resolved, err := resolveTarget(healthTarget)
		if err != nil {
			return err
		}
		return health.Check(resolved)
	},
}

func init() {
	addTargetFlag(healthCmd, &healthTarget)
	rootCmd.AddCommand(healthCmd)
}
