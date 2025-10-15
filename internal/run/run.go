package run

import (
	"io"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/spf13/cobra"
)

func Execute(args []string, stdout, stderr io.Writer) error {
	root := &cobra.Command{
		Use:   "topo",
		Short: "Topo CLI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	listCmd := &cobra.Command{
		Use:   "list-templates",
		Short: "List available templates",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return core.ListTemplates()
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, _ []string) {
			core.PrintVersion()
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
		Use:   "init-project <project-path> <project-name> [ssh-target]",
		Short: "Initialise a new project",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := args[0]
			projectName := args[1]
			sshTarget := ""
			if len(args) == 3 {
				sshTarget = args[2]
			}
			return core.RunInitProject(projectPath, projectName, sshTarget)
		},
	}

	addCmd := &cobra.Command{
		Use:   "add-service <compose-filepath> <template-id> [service-name]",
		Short: "Add a service to the compose file",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFilePath := args[0]
			templateID := args[1]
			serviceName := templateID
			if len(args) == 3 {
				serviceName = args[2]
			}
			return core.RunAddService(composeFilePath, templateID, serviceName, core.CloneProject)
		},
	}

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
		Use:   "generate-makefile <compose-filepath> [ssh-target]",
		Short: "Generate a Makefile for the project",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			composeFilePath := args[0]
			sshTarget := ""
			if len(args) == 2 {
				sshTarget = args[1]
			}
			return core.GenerateMakefile(composeFilePath, sshTarget)
		},
	}

	getContCmd := &cobra.Command{
		Use:   "get-containers-info [ssh-target]",
		Short: "Show container info running on the board",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sshTarget := ""
			if len(args) == 1 {
				sshTarget = args[0]
			}
			return core.GetContainersInfo(sshTarget)
		},
	}

	root.AddCommand(listCmd, versionCmd, cfgCmd, getProjectCmd, initCmd, addCmd, removeCmd, genCmd, getContCmd)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	return root.Execute()
}
