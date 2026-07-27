package main

import (
	"os"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers on the target for the current Compose project.",
	Long: `List containers on the target for the current Compose project.

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Use -f to specify a different compose file.
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName(cmd)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)
		hostName := ssh.NewConfig(dest).HostName
		allContainers, err := cmd.Flags().GetBool("all")
		if err != nil {
			panic("internal error: all flag not registered: " + err.Error())
		}

		host := command.NewHostFromDestination(dest)
		containers, err := docker.ListContainers(composeFile, host, hostName, allContainers)
		if err != nil {
			return err
		}

		return views.Print(newContainerList(containers), os.Stdout, outputFormat)
	},
}

func newContainerList(containers []docker.Container) views.ContainerList {
	items := make([]views.Container, len(containers))
	for i, container := range containers {
		items[i] = views.Container{
			ID:               container.Id,
			Names:            container.Names,
			Image:            container.Image,
			State:            container.State,
			Status:           container.Status,
			ProcessingDomain: container.ProcessingDomain,
			Address:          container.Address,
		}
	}
	return views.ContainerList{Containers: items}
}

func init() {
	addTargetFlag(topoPsCmd)
	addComposeFileFlag(topoPsCmd)
	topoPsCmd.Flags().BoolP("all", "a", false, "show all containers, including stopped")
	rootCmd.AddCommand(topoPsCmd)
}
