package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/fact"
	"github.com/arm/topo/internal/migrate"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
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
  topo configure GREETING_NAME="World"

  # Remove parameters explicitly
  topo configure --unset GREETING_NAME`,
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

		unset, err := cmd.Flags().GetStringArray("unset")
		if err != nil {
			return err
		}

		resolver, err := newConfigurationResolver(cmd, args, unset, os.Stderr)
		if err != nil {
			return err
		}

		changes, err := project.Configure(project.Scope{
			ComposeFile: composeFilePath,
			EnvFiles:    envFiles,
		}, resolver)
		if err != nil || changes == nil {
			return err
		}

		if outputPath == "-" {
			content, err := env.ToString(changes, env.EncodeOptions{})
			if err != nil {
				return fmt.Errorf("failed to encode env file content: %w", err)
			}
			if _, err := fmt.Fprint(os.Stdout, content); err != nil {
				return fmt.Errorf("failed to write env file: %w", err)
			}
			return nil
		}
		return env.UpdateFile(outputPath, changes, env.EncodeOptions{})
	},
}

func init() {
	if experimentalFeaturesEnabled() {
		addTargetFlag(configureCmd)
	}
	configureCmd.Flags().StringArray("unset", nil, "remove a parameter from the output env file; repeat for multiple parameters")
	addNoPromptFlag(configureCmd)
	configureCmd.Flags().StringP("output", "o", env.DefaultFilename, fmt.Sprintf("env file to update, preserving existing entries (default: %s beside the Compose file). Use - to print resolved values to stdout", env.DefaultFilename))
	addComposeFileFlag(configureCmd)
	addEnvFileFlag(configureCmd)
	addMigrateToEnvFlag(configureCmd)
	rootCmd.AddCommand(configureCmd)
}

func newConfigurationResolver(cmd *cobra.Command, assignments, removals []string, output *os.File) (*parameter.StrictResolverChain, error) {
	cli, err := parameter.NewCLIResolver(assignments, removals)
	if err != nil {
		return nil, err
	}
	resolvers := []parameter.Resolver{cli}
	if experimentalFeaturesEnabled() {
		targetResolver := &targetFactsResolver{ctx: cmd.Context(), providers: fact.Providers}
		if target, ok := lookupTarget(cmd); ok {
			targetResolver.target = runner.For(ssh.NewDestination(target.value))
		}
		resolvers = append(resolvers, targetResolver)
	}
	if promptsEnabled(cmd) && term.IsTerminal(output) && term.IsTerminal(os.Stdin) {
		resolvers = append(resolvers, parameter.NewInteractiveResolverForTerminal(os.Stdin, output))
	}
	return parameter.NewStrictResolverChain(resolvers...), nil
}

type targetFactsResolver struct {
	ctx       context.Context
	target    runner.CommandRunner
	providers map[string]fact.Provider
}

func (r *targetFactsResolver) Resolve(parameters []parameter.Parameter) (parameter.Changes, error) {
	if r.target != nil {
		return parameter.NewFactsResolver(r.ctx, r.target, r.providers).Resolve(parameters)
	}
	sourceable, err := parameter.SourceableParameters(parameters, r.providers)
	if err != nil {
		return nil, err
	}
	for _, param := range sourceable {
		logger.Warn(fmt.Sprintf("cannot automatically configure parameter %q without a target; provide --target or set TOPO_TARGET, or supply this parameter manually", param.Name))
	}
	return nil, nil
}
