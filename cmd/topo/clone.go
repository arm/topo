package main

import (
	"os"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/spf13/cobra"
)

var topoCloneCmd = &cobra.Command{
	Use:   "clone <project-source> [<path>]",
	Short: "Clone a Project",
	Long: `Clone a Project to the specified path.

The project-source argument uses scheme prefixes to specify the source type.
The git: prefix is optional for git@host and https:// URLs.

Some projects require parameters. Supply them on the command line or answer
interactive prompts.`,
	Example: `  # Git repository
  topo clone git@github.com:user/repo.git
  topo clone https://github.com/user/repo.git#develop
  topo clone git:git@github.com:user/repo.git
  topo clone git:https://github.com/user/repo.git#main
  topo clone git:ubuntu@example.com:repo.git
  topo clone git:builder@host:tools/platform.git#v2

  # Local directory (must contain a Topo Project)
  topo clone dir:/path/to/project/folder
  topo clone dir:./relative/path

  # Will prompt for required parameters
  topo clone https://github.com/Arm-Examples/topo-welcome.git

  # Provide parameters explicitly
  topo clone https://github.com/Arm-Examples/topo-welcome.git GREETING_NAME="World"

  # With an explicit path
  topo clone https://github.com/Arm-Examples/topo-welcome.git my-demo GREETING_NAME="World"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		src := args[0]

		projectSource, err := project.NewSource(src)
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

		var resolvers []parameter.Resolver
		if len(cliArgs) > 0 {
			cliResolver, err := parameter.NewCLIResolver(cliArgs)
			if err != nil {
				return err
			}
			resolvers = append(resolvers, cliResolver)
		}
		if term.IsTTY(os.Stdout) && term.IsTTY(os.Stdin) {
			resolvers = append(resolvers, parameter.NewInteractiveResolver(os.Stdin, os.Stdout))
		}

		resolver := parameter.NewStrictResolverChain(resolvers...)

		return project.NewClone(path, projectSource, resolver).Run(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(topoCloneCmd)
}
