package main

import (
	"os"
	"strings"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/template"
	"github.com/spf13/cobra"
)

var topoCloneCmd = &cobra.Command{
	Use:   "clone <project-source> [<path>]",
	Short: "Clone an example project",
	Long: `Clone an example project to the specified path.

The project-source argument uses scheme prefixes to specify the source type.
The git: prefix is optional for git@host and https:// URLs.

Some projects require build arguments. Supply them on the command line or answer
interactive prompts.`,
	Example: `  # Git repository
  topo clone git@github.com:user/repo.git
  topo clone https://github.com/user/repo.git#develop
  topo clone git:git@github.com:user/repo.git
  topo clone git:https://github.com/user/repo.git#main
  topo clone git:ubuntu@example.com:repo.git
  topo clone git:builder@host:tools/platform.git#v2

  # Local directory (must contain a Topo template)
  topo clone dir:/path/to/template/folder
  topo clone dir:./relative/path

  # Will prompt for required args
  topo clone https://github.com/Arm-Examples/topo-welcome.git

  # Provide build arguments explicitly
  topo clone https://github.com/Arm-Examples/topo-welcome.git GREETING_NAME="World"

  # With an explicit path
  topo clone https://github.com/Arm-Examples/topo-welcome.git my-demo GREETING_NAME="World"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		src := args[0]

		projectSource, err := template.NewSource(src)
		if err != nil {
			return err
		}

		var path string
		var cliArgs []string
		if len(args) >= 2 && !strings.Contains(args[1], "=") {
			path = args[1]
			cliArgs = args[2:]
		} else {
			path, err = projectSource.GetName()
			if err != nil {
				return err
			}
			cliArgs = args[1:]
		}

		var providers []arguments.Provider
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

		return project.NewClone(path, projectSource, argProvider).Run(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(topoCloneCmd)
}
