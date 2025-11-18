package main

import (
	"os"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/spf13/cobra"
)

var addServiceNoPrompt bool

var addServiceCmd = &cobra.Command{
	Use:   "add-service <compose-filepath> <service-name> <source> [flags] [-- ARG=VALUE ...]",
	Short: "Add a service to the compose file from a template ID or git URL",
	Long: `Add a service to the compose file.

The source argument uses scheme prefixes to specify the source type:

Template ID (from built-in templates):
  topo add-service compose.yaml my-service template:hello-world

Git repository:
  topo add-service compose.yaml my-service git:git@github.com:user/repo.git
  topo add-service compose.yaml my-service git:https://github.com/user/repo.git#develop
  topo add-service compose.yaml my-service git:git@github.com:user/repo.git#main

Service templates may require build arguments. You can provide them via the command line
or interactively when prompted:

  # Will prompt for required args
  topo add-service compose.yaml my-service git:url
  # Provide args explicitly
  topo add-service compose.yaml my-service git:url -- GREETING="Hello" PORT=8080
  # Don't prompt, raise an error if required args are missing
  topo add-service compose.yaml my-service git:url --no-prompt

Use list-service-templates to see available built-in templates.`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		serviceName := args[1]
		sourceArg := args[2]

		src, err := source.Parse(sourceArg)
		if err != nil {
			return err
		}

		var providers []arguments.Provider
		var cliArgs []string
		if len(args) > 3 {
			cliArgs = args[3:]
		}
		if len(cliArgs) > 0 {
			cliProvider, err := arguments.NewCLIProvider(cliArgs)
			if err != nil {
				return err
			}
			providers = append(providers, cliProvider)
		}
		if !addServiceNoPrompt {
			providers = append(providers, arguments.NewInteractiveProvider(os.Stdin, os.Stdout))
		}

		argCollector := arguments.NewCollector(providers...)

		return core.AddService(composeFilePath, serviceName, src, argCollector)
	},
}

func init() {
	addServiceCmd.Flags().BoolVar(&addServiceNoPrompt, "no-prompt", false, "Error if required arguments are missing instead of prompting")
	rootCmd.AddCommand(addServiceCmd)
}
