package main

import (
	"os"

	"github.com/arm/topo/internal/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise a new project in the current directory",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		workDir, err := os.Getwd()
		if err != nil {
			return err
		}
		return project.Init(workDir)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
