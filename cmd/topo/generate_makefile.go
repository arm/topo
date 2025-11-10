package main

import (
	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

var generateMakefileTarget string

var generateMakefileCmd = &cobra.Command{
	Use:   "generate-makefile <compose-filepath>",
	Short: "Generate a Makefile for the project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		composeFilePath := args[0]
		resolved, err := core.ResolveTarget(generateMakefileTarget)
		if err != nil {
			return err
		}
		return core.GenerateMakefile(composeFilePath, resolved)
	},
}

func init() {
	addTargetFlag(generateMakefileCmd, &generateMakefileTarget)
	rootCmd.AddCommand(generateMakefileCmd)
}
