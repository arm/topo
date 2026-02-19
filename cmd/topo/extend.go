package main

import (
	"os"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/template"
	"github.com/spf13/cobra"
)

var extendCmd = &cobra.Command{
	Use:   "extend <compose-filepath> <source> [flags] [-- ARG=VALUE ...]",
	Short: "Add all services of source to the compose file from a template ID, git URL, or local directory",
	Long: `Add all services of source to the compose file.

The source argument uses scheme prefixes to specify the source type:

Template ID (from built-in templates):
  topo extend compose.yaml template:Topo-Welcome

Git repository (git: prefix is optional for git@host and https:// URLs):
  topo extend compose.yaml git:https://github.com/user/repo.git
  topo extend compose.yaml https://github.com/user/repo.git
  topo extend compose.yaml https://github.com/user/repo.git#develop
  topo extend compose.yaml git:git@github.com:user/repo.git
  topo extend compose.yaml git@github.com:user/repo.git
  topo extend compose.yaml git@github.com:user/repo.git#main
  topo extend compose.yaml git:ubuntu@example.com:repo.git
  topo extend compose.yaml git:builder@host:tools/platform.git#v2

Local directory:
  topo extend compose.yaml dir:/path/to/template/folder
  topo extend compose.yaml dir:./relative/path

Service templates may require build arguments. You can provide them via the command line
or interactively when prompted:

  # Will prompt for required args
  topo extend compose.yaml git:url
  # Provide args explicitly
  topo extend compose.yaml git:url -- GREETING="Hello" PORT=8080
`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		sourceArg := args[1]

		outputFormat, err := resolveOutput(cmd)
		if err != nil {
			return err
		}
		c := console.NewLogger(os.Stderr, outputFormat)

		if err != nil {
			return err
		}

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

		logs, err := project.Extend(composeFilePath, src, argProvider)

		c.Log(logs...)
		return err
	},
}

func init() {
	rootCmd.AddCommand(extendCmd)
}
