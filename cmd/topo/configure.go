package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/env"
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
the interactive parameter browser. Search for a parameter, stage edits, then
press Ctrl+S to save. Escape cancels without writing.`,
	Example: `  # Browse, search, and edit project parameters
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

		if migrateToEnv(cmd) {
			err := migrate.ToEnv(composeFilePath)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("successfully migrated %q to be parameterized from %q", composeFilePath, env.DefaultFilename))
			return nil
		}

		envFiles, err := getEnvFiles(cmd, composeFilePath)
		if err != nil {
			return err
		}

		usesLiteralBuildArgs, err := migrate.UsesLiteralBuildArgConfiguration(composeFilePath)
		if err != nil {
			return err
		}
		if usesLiteralBuildArgs {
			return fmt.Errorf("this project appears to use the parameter format supported by Topo versions older than 14.0.0. Try running 'topo configure --migrate-to-env', then retry configuration")
		}

		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("output") {
			outputPath = filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename)
		}

		var resolvers []parameter.Resolver
		if len(args) > 0 {
			cliResolver, err := parameter.NewCLIResolver(args)
			if err != nil {
				return err
			}
			resolvers = append(resolvers, cliResolver)
		}
		interactive := term.IsTTY(os.Stderr) && term.IsTTY(os.Stdin)
		if interactive {
			resolvers = append(resolvers, &parameter.BrowserResolver{
				Input: os.Stdin, Output: os.Stderr,
				EnvFiles: envFiles, Destination: outputPath,
			})
		}

		resolver := parameter.NewStrictResolverChain(resolvers...)

		values, err := project.Configure(project.Scope{
			ComposeFile: composeFilePath,
			EnvFiles:    envFiles,
		}, resolver)
		if err != nil {
			return err
		}
		if values == nil {
			if interactive {
				_, err = fmt.Fprintln(os.Stderr, "No changes written.")
			}
			return err
		}

		if outputPath == "-" {
			content, err := env.ToString(values, env.EncodeOptions{})
			if err != nil {
				return fmt.Errorf("failed to encode env file content: %w", err)
			}
			if _, err := fmt.Fprint(os.Stdout, content); err != nil {
				return fmt.Errorf("failed to write env file: %w", err)
			}
			return nil
		}
		if err := env.UpdateFile(outputPath, values, env.EncodeOptions{}); err != nil {
			return err
		}
		if interactive {
			_, err = fmt.Fprintf(os.Stderr, "Saved %d parameter(s) to %s.\n", len(values), outputPath)
		}
		return err
	},
}

func init() {
	configureCmd.Flags().StringP("output", "o", env.DefaultFilename, fmt.Sprintf("env file to update, preserving existing entries (default: %s beside the Compose file). Use - to print resolved values to stdout", env.DefaultFilename))
	addComposeFileFlag(configureCmd)
	addEnvFileFlag(configureCmd)
	addMigrateToEnvFlag(configureCmd)
	rootCmd.AddCommand(configureCmd)
}
