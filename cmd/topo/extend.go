package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/spf13/cobra"
)

var extendNoPrompt bool

var extendCmd = &cobra.Command{
	Use:   "extend <compose-filepath> <source> [flags] [-- ARG=VALUE ...]",
	Short: "Add all services of source to the compose file from a template ID, git URL, or local directory",
	Long: `Add all services of source to the compose file.

The source argument uses scheme prefixes to specify the source type:

Template ID (from built-in templates):
  topo extend compose.yaml template:hello-world

Git repository:
  topo extend compose.yaml git:git@github.com:user/repo.git
  topo extend compose.yaml git:https://github.com/user/repo.git#develop
  topo extend compose.yaml git:git@github.com:user/repo.git#main

Local directory:
  topo extend compose.yaml dir:/path/to/template/folder
  topo extend compose.yaml dir:./relative/path

Service templates may require build arguments. You can provide them via the command line
or interactively when prompted:

  # Will prompt for required args
  topo extend compose.yaml git:url
  # Provide args explicitly
  topo extend compose.yaml git:url -- GREETING="Hello" PORT=8080
  # Don't prompt, raise an error if required args are missing
  topo extend compose.yaml git:url --no-prompt
`,
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
		if !extendNoPrompt {
			providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))
		}

		argProvider := arguments.NewStrictProviderChain(providers...)

		return project.Extend(composeFilePath, src, argProvider)
	},
}

func init() {
	extendCmd.Flags().BoolVar(&extendNoPrompt, "no-prompt", false, "Error if required arguments are missing instead of prompting")
	rootCmd.AddCommand(extendCmd)
}
