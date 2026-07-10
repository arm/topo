package main

import (
	"os"

	"github.com/arm/topo/cli/internal/arguments"
	"github.com/arm/topo/cli/internal/output/term"
	"github.com/arm/topo/cli/internal/project"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure [PARAMETER=VALUE ...]",
	Short: "Configure project parameters",
	Long: `Configure project parameters for the Topo project in the current directory.

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Use -f to specify a different compose file.

Some projects require parameters. Supply them on the command line or answer
interactive prompts.`,
	Example: `  # Will prompt for required parameters
  topo configure

  # Provide parameters explicitly
  topo configure GREETING_NAME="World"`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		composeFile, err := getComposeFileName(cmd)
		if err != nil {
			return err
		}

		var providers []arguments.Provider
		if len(args) > 0 {
			cliProvider, err := arguments.NewCLIProvider(args)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		if term.IsTTY(os.Stdout) && term.IsTTY(os.Stdin) {
			providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))
		}

		parameterProvider := arguments.NewStrictProviderChain(providers...)

		return project.ResolveAndApplyArgs(composeFile, parameterProvider)
	},
}

func init() {
	addComposeFileFlag(configureCmd)
	rootCmd.AddCommand(configureCmd)
}
