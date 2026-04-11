package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/vscode"
	"github.com/spf13/cobra"
)

var getProjectCmd = &cobra.Command{
	Use:    "get-project <compose-filepath>",
	Short:  "Print the project as JSON",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		composeFilePath := args[0]
		return vscode.PrintProject(os.Stdout, composeFilePath)
	},
}

var listCandidateTargets = &cobra.Command{
	Use:    "list-candidate-targets <config-filepath>",
	Short:  "Prints a list of candidate ssh targets defined in the given config file as JSON",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		configPath := args[0]

		hosts := ssh.ListHosts(configPath)
		data, err := json.Marshal(hosts)
		if err != nil {
			return fmt.Errorf("failed to marshal ssh hosts: %w", err)
		}
		_, err = fmt.Fprintf(os.Stdout, "%s\n", data)
		return err
	},
}

func init() {
	rootCmd.AddCommand(getProjectCmd)
	rootCmd.AddCommand(listCandidateTargets)
}
