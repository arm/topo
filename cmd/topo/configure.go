package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/migrate"
	"github.com/arm/topo/internal/output/logger"
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

		composeFilePath, err := resolveComposeFilePath(cmd)
		if err != nil {
			return err
		}

		outputEnvFile, err := resolveOutputEnvFile(cmd, filepath.Dir(composeFilePath), ".")
		if err != nil {
			return err
		}

		if migrateToEnv(cmd) {
			err := migrate.ToEnv(composeFilePath, outputEnvFile)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("successfully migrated %q to be parameterized from %q", composeFilePath, outputEnvFile))
			return nil
		}

		usesLiteralBuildArgs, err := migrate.UsesLiteralBuildArgConfiguration(composeFilePath)
		if err != nil {
			return err
		}
		if usesLiteralBuildArgs {
			return fmt.Errorf("this project appears to use the parameter format supported by Topo versions older than 14.0.0. Try running 'topo configure --migrate-to-env', then retry configuration")
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

		return project.Configure(composeFilePath, outputEnvFile, resolver)
	},
}

func init() {
	addComposeFileFlag(configureCmd)
	addMigrateToEnvFlag(configureCmd)
	addOutputEnvFileFlag(configureCmd, "the current working directory")
	rootCmd.AddCommand(configureCmd)
}
