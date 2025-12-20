package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/spf13/cobra"
)

var topoCloneCmd = &cobra.Command{
	Use:   "clone <path> <project-source>",
	Short: "Clone an example project",
	Long: `Clone an example project to the specified path.

The project-source argument uses scheme prefixes to specify the source type:

Template ID (from built-in catalog):
  topo clone my-demo template:Topo-Welcome

Git repository:
  topo clone my-demo git:git@github.com:user/repo.git
  topo clone my-demo git:https://github.com/user/repo.git#develop
  topo clone my-demo git:git@github.com:user/repo.git#main

Local directory (must contain a Topo template):
  topo clone my-demo dir:/path/to/template/folder
  topo clone my-demo dir:./relative/path

Some projects require build arguments. Supply them on the command line or answer prompts:

  # Will prompt for required args
  topo clone my-demo template:Topo-Welcome
  # Provide args explicitly
  topo clone my-demo template:Topo-Welcome GREETING="Hello" PORT=8080
`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		path := args[0]
		src := args[1]

		var providers []arguments.Provider
		var cliArgs []string
		if len(args) > 2 {
			cliArgs = args[2:]
		}
		if len(cliArgs) > 0 {
			cliProvider, err := arguments.NewCLIProvider(cliArgs)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))

		argProvider := arguments.NewStrictProviderChain(providers...)

		projectSource, err := template.NewSource(src)
		if err != nil {
			return err
		}

		return project.Clone(path, projectSource, argProvider, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(topoCloneCmd)
}
