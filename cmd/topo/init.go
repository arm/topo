package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var initTarget string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise a new project in the current directory",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := os.Getwd()
		if err != nil {
			return err
		}
		resolved, err := core.ResolveTarget(initTarget)
		if err != nil {
			return err
		}
		return core.InitProject(workDir, resolved)
	},
}

func init() {
	addTargetFlag(initCmd, &initTarget)
	rootCmd.AddCommand(initCmd)
}
