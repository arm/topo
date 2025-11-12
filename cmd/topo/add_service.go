package main

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/spf13/cobra"
)

var addServiceGitRef string

var addServiceCmd = &cobra.Command{
	Use:   "add-service <compose-filepath> <service-name> <source>",
	Short: "Add a service to the compose file from a template ID or git URL",
	Long: `Add a service to the compose file.

The source argument uses scheme prefixes to specify the source type:

Template ID (from built-in templates):
  topo add-service compose.yaml my-service template:hello-world

Git repository:
  topo add-service compose.yaml my-service git:git@github.com:user/repo.git
  topo add-service compose.yaml my-service git:https://github.com/user/repo.git --ref develop

Use list-service-templates to see available built-in templates.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		composeFilePath := args[0]
		serviceName := args[1]
		sourceArg := args[2]

		sourceType, sourceValue, err := source.Parse(sourceArg)
		if err != nil {
			return err
		}

		switch sourceType {
		case "template":
			src := source.TemplateId(sourceValue)
			return core.AddService(composeFilePath, serviceName, src)
		case "git":
			src := source.Git{
				URL: sourceValue,
				Ref: addServiceGitRef,
			}
			return core.AddService(composeFilePath, serviceName, src)
		default:
			return fmt.Errorf("unsupported source type: %s (supported: template:, git:)", sourceType)
		}
	},
}

func init() {
	addServiceCmd.Flags().StringVar(&addServiceGitRef, "ref", "", "Git branch or ref to checkout (use with git: sources)")
	rootCmd.AddCommand(addServiceCmd)
}
