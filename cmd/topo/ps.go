package main

import (
	"os"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running containers on the target for the current Compose project.",
	Long: `List running containers on the target for the current Compose project.

The compose.yaml must be in the current working directory, as this is used to select containers to be viewed.
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName()
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)
		hostName := ssh.NewConfig(dest).HostName
		containers, err := deploy.ListRunningContainers(composeFile, command.NewHostFromDestination(dest), hostName)
		if err != nil {
			return err
		}

		return views.Print(views.PrintablePSReport{Containers: containers}, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(topoPsCmd)
	rootCmd.AddCommand(topoPsCmd)
}
