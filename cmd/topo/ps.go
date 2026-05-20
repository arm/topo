package main

import (
	"os"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List project containers running on the target",
	Long: `List running containers on the target host for the current Compose project.

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
		containers, err := deploy.ListRunningContainers(composeFile, command.NewHostFromDestination(dest))
		if err != nil {
			return err
		}

		hostName := ssh.NewConfig(dest).HostName
		rows := make([]templates.ContainerStatus, len(containers))
		for i, c := range containers {
			rows[i] = templates.ContainerStatus{
				Image:   c.Image,
				Status:  c.Status,
				Address: deploy.PublishedAddress(c.Ports, hostName),
			}
		}

		return printable.Print(templates.PrintablePSReport{Containers: rows}, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(topoPsCmd)
	rootCmd.AddCommand(topoPsCmd)
}
