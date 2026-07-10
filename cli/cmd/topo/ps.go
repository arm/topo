package main

import (
	"os"

	"github.com/arm/topo/cli/internal/deploy"
	"github.com/arm/topo/cli/internal/deploy/command"
	"github.com/arm/topo/cli/internal/output/views"
	"github.com/arm/topo/cli/internal/ssh"
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
		containers, err := deploy.ListContainers(composeFile, host, hostName, allContainers)
		if err != nil {
			return err
		}

		return views.Print(views.ContainerList{Containers: containers}, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(topoPsCmd)
	addComposeFileFlag(topoPsCmd)
	topoPsCmd.Flags().BoolP("all", "a", false, "show all containers, including stopped")
	rootCmd.AddCommand(topoPsCmd)
}
