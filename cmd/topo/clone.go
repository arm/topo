package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/spf13/cobra"
)

var topoCloneCmd = &cobra.Command{
	Use:   "clone <path> <project-source>",
	Short: "Clone an example project",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		path := args[0]
		src := args[1]

		var providers []arguments.Provider
		var cliArgs []string
		if len(args) > 2 {
			cliArgs = args[2:]
		}
		if len(cliArgs) > 0 {
			cliProvider, err := arguments.NewCLIProvider(cliArgs)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))

		argProvider := arguments.NewStrictProviderChain(providers...)

		projectSource, err := template.NewSource(src)
		if err != nil {
			return err
		}

		return project.Clone(path, projectSource, argProvider, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(topoCloneCmd)
}
