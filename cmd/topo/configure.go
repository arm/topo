package main

import (
	"os"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
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

		var providers []parameter.Provider
		if len(args) > 0 {
			cliProvider, err := parameter.NewCLIProvider(args)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		if term.IsTTY(os.Stdout) && term.IsTTY(os.Stdin) {
			providers = append(providers, parameter.NewInteractiveProvider(os.Stdin, os.Stdout))
		}

		provider := parameter.NewStrictProviderChain(providers...)

		return project.Configure(composeFile, provider)
	},
}

func init() {
	addComposeFileFlag(configureCmd)
	rootCmd.AddCommand(configureCmd)
}
