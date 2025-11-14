package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var getContainersInfoTarget string

var getContainersInfoCmd = &cobra.Command{
	Use:   "get-containers-info",
	Short: "Show container info running on the board",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolved, err := resolveTarget(getContainersInfoTarget)
		if err != nil {
			return err
		}
		return core.PrintContainersInfo(os.Stdout, resolved)
	},
}

func init() {
	addTargetFlag(getContainersInfoCmd, &getContainersInfoTarget)
	rootCmd.AddCommand(getContainersInfoCmd)
}
