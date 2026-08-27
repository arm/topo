package main

import (
	"os"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var psEngine string

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

		parsedEngine, err := parseContainerEngine(psEngine)
		if err != nil {
			return err
		}
		targetArg, err := requireTarget(cmd)
		if err != nil {
			return err
		}

		composeFile, err := getComposeFileName(cmd)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(targetArg)
		hostname, err := ssh.ResolveHostname(dest)
		if err != nil {
			return err
		}
		allContainers, err := cmd.Flags().GetBool("all")
		if err != nil {
			panic("internal error: all flag not registered: " + err.Error())
		}

		if parsedEngine == containerEnginePodman {
			containers, err := podman.ListContainers(composeFile, dest, hostname, allContainers)
			if err != nil {
				return err
			}
			return views.Print(newPodmanContainerList(containers), os.Stdout, outputFormat)
		}

		host := docker.NewHostFromDestination(dest)
		containers, err := docker.ListContainers(composeFile, host, hostname, allContainers)
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

func newPodmanContainerList(containers []podman.Container) views.ContainerList {
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
	if experimentalFeaturesEnabled() {
		topoPsCmd.Flags().StringVar(&psEngine, "engine", string(containerEngineDocker), "container engine to use (docker or podman)")
	}
	rootCmd.AddCommand(topoPsCmd)
}
