package main

import (
	"os"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/template"
	"github.com/spf13/cobra"
)

var extendCmd = &cobra.Command{
	Use:   "extend <compose-filepath> <source> [flags] [-- ARG=VALUE ...]",
	Short: "Add services from a template to the compose file",
	Long: `Add all services from a source to the compose file.

The source argument uses scheme prefixes to specify the source type.
The git: prefix is optional for git@host and https:// URLs.

Service templates may require build arguments. You can provide them after --
or answer interactive prompts.`,
	Example: `  # Git repository
  topo extend compose.yaml https://github.com/user/repo.git
  topo extend compose.yaml https://github.com/user/repo.git#develop
  topo extend compose.yaml git:git@github.com:user/repo.git
  topo extend compose.yaml git:builder@host:tools/platform.git#v2

  # Local directory
  topo extend compose.yaml dir:/path/to/template/folder
  topo extend compose.yaml dir:./relative/path

  # Provide build arguments explicitly
  topo extend compose.yaml git:url -- GREETING="Hello" PORT=8080`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		sourceArg := args[1]

		src, err := template.NewSource(sourceArg)
		if err != nil {
			return err
		}

		var providers []arguments.Provider
		var cliArgs []string
		if dashIdx := cmd.ArgsLenAtDash(); dashIdx >= 0 {
			cliArgs = args[dashIdx:]
		}
		if len(cliArgs) > 0 {
			cliProvider, err := arguments.NewCLIProvider(cliArgs)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		if term.IsTTY(os.Stdout) && term.IsTTY(os.Stdin) {
			providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))
		}

		argProvider := arguments.NewStrictProviderChain(providers...)

		return project.Extend(composeFilePath, src, argProvider)
	},
}

func init() {
	rootCmd.AddCommand(extendCmd)
}
