package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/ssh"
	"github.com/spf13/cobra"
)

var topoPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List services in a currently running deployment",
	Long: `List services that are already running on the target host using definitions in the compose file.

The compose file (compose.yaml) must be in the current working directory, as this is used to select containers to be viewed.
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

		dest := command.NewHostFromDestination(ssh.NewDestination(targetArg))

		ps := operation.NewDockerComposePs(composeFile, dest)

		var rawPsOut bytes.Buffer

		if err := ps.Run(&rawPsOut); err != nil {
			return err
		}

		decoder := json.NewDecoder(&rawPsOut)
		var containers []templates.ContainerStatus
		for decoder.More() {
			var container templates.ContainerStatus
			if err := decoder.Decode(&container); err != nil {
				return err
			}
			if i := strings.Index(container.Ports, "->"); i != -1 {
				container.Ports = container.Ports[:i]
			}
			container.Ports = strings.ReplaceAll(container.Ports, "0.0.0.0", ssh.NewConfig(ssh.NewDestination(targetArg)).HostName)
			containers = append(containers, container)
		}

		toPrint := templates.PrintablePSReport{Containers: containers}

		return printable.Print(toPrint, os.Stdout, outputFormat)
	},
}

func init() {
	addTargetFlag(topoPsCmd)
	rootCmd.AddCommand(topoPsCmd)
}
