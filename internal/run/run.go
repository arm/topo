package run

import (
	"fmt"
	"io"
	"os"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/arm-debug/topo-cli/internal/version"
	"github.com/spf13/cobra"
)

func addTargetFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "target", "", "The SSH destination.")
}

func Execute(args []string, stdout, stderr io.Writer) error {
	var target string

	root := &cobra.Command{
		Use:     "topo",
		Short:   "Topo CLI",
		Version: fmt.Sprintf("%s (commit: %s)", version.Version, version.GitCommit),
	}

	listCmd := &cobra.Command{
		Use:   "list-service-templates",
		Short: "List available Service Templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return template.List()
		},
	}

	cfgCmd := &cobra.Command{
		Use:   "get-config-metadata",
		Short: "Show config metadata",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return core.GetConfigMetadata()
		},
	}

	getProjectCmd := &cobra.Command{
		Use:   "get-project <compose-filepath>",
		Short: "Print the project as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFilePath := args[0]
			return core.GetProject(composeFilePath)
		},
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialise a new project in the current directory",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return err
			}
			resolved, err := core.ResolveTarget(target)
			if err != nil {
				return err
			}
			return core.RunInitProject(workDir, resolved)
		},
	}
	addTargetFlag(initCmd, &target)

	var gitRef string

	addCmd := &cobra.Command{
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
				return core.RunAddServiceByTemplateId(composeFilePath, sourceValue, serviceName, core.CloneProject, template.GetTemplate)
			case "git":
				return core.RunAddService(composeFilePath, sourceValue, gitRef, serviceName, core.CloneProject)
			default:
				return fmt.Errorf("unsupported source type: %s (supported: template:, git:)", sourceType)
			}
		},
	}
	addCmd.Flags().StringVar(&gitRef, "ref", "", "Git branch or ref to checkout (use with git: sources)")

	removeCmd := &cobra.Command{
		Use:   "remove-service <compose-filepath> <service-name>",
		Short: "Remove a service from the compose file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFilePath := args[0]
			serviceName := args[1]
			return core.RunRemoveService(composeFilePath, serviceName)
		},
	}

	genCmd := &cobra.Command{
		Use:   "generate-makefile <compose-filepath>",
		Short: "Generate a Makefile for the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFilePath := args[0]
			resolved, err := core.ResolveTarget(target)
			if err != nil {
				return err
			}
			return core.GenerateMakefile(composeFilePath, resolved)
		},
	}
	addTargetFlag(genCmd, &target)

	getContCmd := &cobra.Command{
		Use:   "get-containers-info",
		Short: "Show container info running on the board",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := core.ResolveTarget(target)
			if err != nil {
				return err
			}
			return core.GetContainersInfo(resolved)
		},
	}
	addTargetFlag(getContCmd, &target)

	healthCmd := &cobra.Command{
		Use:   "check-health",
		Short: "Show information about the target and check the host environment (container engines, SSH availability)",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := core.ResolveTarget(target)
			if err != nil {
				return err
			}
			return core.CheckHealth(resolved)
		},
	}
	addTargetFlag(healthCmd, &target)

	root.AddCommand(listCmd, cfgCmd, getProjectCmd, initCmd, addCmd, removeCmd, genCmd, getContCmd, healthCmd)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	return root.Execute()
}
