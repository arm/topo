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

		var resolvers []parameter.Resolver
		if len(args) > 0 {
			cliResolver, err := parameter.NewCLIResolver(args)
			if err != nil {
				return err
			}
			resolvers = append(resolvers, cliResolver)
		}
		if term.IsTTY(os.Stdout) && term.IsTTY(os.Stdin) {
			resolvers = append(resolvers, parameter.NewInteractiveResolver(os.Stdin, os.Stdout))
		}

		resolver := parameter.NewStrictResolverChain(resolvers...)

		return project.Configure(composeFile, resolver)
	},
}

func init() {
	addComposeFileFlag(configureCmd)
	rootCmd.AddCommand(configureCmd)
}
