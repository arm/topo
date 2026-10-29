package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers on the target for the current Compose project.",
	Long: `List containers on the target for the current Compose project.

By default, Topo uses compose.yaml in the current working directory, then compose.yml. Repeat -f to merge Compose files in the specified order.
`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		outputFormat := resolveOutput(cmd)

		engine, err := getEngineSelection(cmd)
		if err != nil {
			return err
		}
		target, err := requireTarget(cmd)
		if err != nil {
			return err
		}
		composeFilePaths, err := resolveComposeFilePaths(cmd)
		if err != nil {
			return err
		}
		envFiles, err := getEnvFiles(cmd, composeFilePaths[0])
		if err != nil {
			return err
		}
		scope, err := project.BuildScope(composeFilePaths, target.value, envFiles)
		if err != nil {
			return err
		}

		dest := ssh.NewDestination(target.value)
		hostname, err := ssh.ResolveHostname(dest)
		if err != nil {
			return err
		}
		allContainers, err := cmd.Flags().GetBool("all")
		if err != nil {
			panic("internal error: all flag not registered: " + err.Error())
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		containers, err := deploy.ListContainers(
			ctx, scope, dest, deploy.EngineKind(engine.value), hostname, allContainers,
		)
		if err != nil {
			return err
		}
		return views.Print(newContainerList(containers), os.Stdout, outputFormat)
	},
}

func newContainerList(containers []deploy.EnrichedContainer) views.ContainerList {
	items := make([]views.Container, len(containers))
	for i, container := range containers {
		items[i] = views.Container(container)
	}
	return views.ContainerList{Containers: items}
}

func init() {
	addTargetFlag(topoPsCmd)
	addComposeFileFlag(topoPsCmd)
	addEnvFileFlag(topoPsCmd)
	topoPsCmd.Flags().BoolP("all", "a", false, "show all containers, including stopped")
	addEngineFlag(topoPsCmd)
	rootCmd.AddCommand(topoPsCmd)
}
